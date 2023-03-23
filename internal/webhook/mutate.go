package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"

	"gomodules.xyz/jsonpatch/v2"

	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/webhook-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io

var (
	logger = ctrl.Log.WithName("mutator")

	imageMutation = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hcw",
		Subsystem: "mutations",
		Name:      "image_rewrite",
		Help:      "image rewrite mutation",
	}, []string{"container_name", "kind", "original_image", "rewritten_image"})
)

func init() {
	metrics.Registry.MustRegister(imageMutation)
}

// ContainerTransformer rewrites docker image references for harbor proxy cache projects.
type ContainerTransformer interface {
	// RewriteImage takes a docker image reference and returns the same image reference rewritten for a harbor
	// proxy cache project endpoint, if one is available, else returns the original image reference.
	RewriteImage(imageRef, platformArch, os string) (string, error)
}

// PodContainerProxier mutates init containers and containers to redirect them to the harbor proxy cache if one exists.
type PodContainerProxier struct {
	Client      client.Client
	Decoder     *admission.Decoder
	Transformer ContainerTransformer
	Verbose     bool
}

// Handle mutates init containers and containers.
func (p *PodContainerProxier) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := p.Decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	platformArch := runtime.GOARCH
	os := runtime.GOOS
	nodeName := pod.Spec.NodeName
	if nodeName != "" {
		platformArch, os, err = lookupNodeArchAndOS(ctx, nodeName)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
	}

	initContainers, updatedInit, err := p.updateContainers(pod.Spec.InitContainers, platformArch, os, "init")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	containers, updated, err := p.updateContainers(pod.Spec.Containers, platformArch, os, "normal")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if !updated && !updatedInit {
		return admission.Allowed("no updates")
	}
	pod.Spec.InitContainers = initContainers
	pod.Spec.Containers = containers

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if p.Verbose {
		patch, err := jsonpatch.CreatePatch(req.Object.Raw, marshaledPod)
		if err == nil { // errors will be surfaced in return below
			logger.Info(fmt.Sprintf("patch for %s: %v", string(pod.UID), patch))
		}
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func lookupNodeArchAndOS(ctx context.Context, nodeName string) (platform, os string, err error) {
	restCfg, err := config.GetConfig()
	if err != nil {
		return "", "", fmt.Errorf("failed to create rest config: %w", err)
	}
	restClient, err := client.New(restCfg, client.Options{})
	if err != nil {
		return "", "", fmt.Errorf("failed to create rest client: %w", err)
	}
	var node *corev1.Node
	if err = restClient.Get(ctx, client.ObjectKey{Name: nodeName}, node); err != nil {
		return "", "", fmt.Errorf("failed to lookup node %s: %w", nodeName, err)
	}
	return node.Status.NodeInfo.Architecture, node.Status.NodeInfo.OperatingSystem, nil
}

func (p *PodContainerProxier) updateContainers(containers []corev1.Container, platform, os, kind string) ([]corev1.Container, bool, error) {
	containersReplacement := make([]corev1.Container, 0, len(containers))
	updated := false
	for i := range containers {
		container := containers[i]
		imageRef, err := p.Transformer.RewriteImage(container.Image, platform, os)
		if err != nil {
			return []corev1.Container{}, false, err
		}
		if !updated {
			updated = imageRef != container.Image
		}
		if imageRef != container.Image {
			logger.Info(fmt.Sprintf("rewriting the image of %q from %q to %q", container.Name, container.Image, imageRef))
			imageMutation.WithLabelValues(container.Name, kind, container.Image, imageRef).Inc()
		}
		container.Image = imageRef
		containersReplacement = append(containersReplacement, container)
	}
	return containersReplacement, updated, nil
}

// PodContainerProxier implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (p *PodContainerProxier) InjectDecoder(d *admission.Decoder) error {
	p.Decoder = d
	return nil
}
