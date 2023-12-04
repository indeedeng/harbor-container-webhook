package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/prometheus/client_golang/prometheus"

	"gomodules.xyz/jsonpatch/v2"

	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

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

// PodContainerProxier mutates init containers and containers to redirect them to the harbor proxy cache if one exists.
type PodContainerProxier struct {
	Client       client.Client
	Decoder      *admission.Decoder
	Transformers []ContainerTransformer
	Verbose      bool

	// kube config settings
	KubeClientBurst int
	KubeClientQPS   float32
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
		platformArch, os, err = p.lookupNodeArchAndOS(ctx, p.Client, nodeName)
		if err != nil {
			logger.Info(fmt.Sprintf("unable to lookup node for pod %q, defaulting pod to webhook runtime OS and architecture: %s", string(pod.UID), err.Error()))
		}
	}

	initContainers, updatedInit, err := p.updateContainers(ctx, pod.Spec.InitContainers, platformArch, os, "init")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	containers, updated, err := p.updateContainers(ctx, pod.Spec.Containers, platformArch, os, "normal")
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

func (p *PodContainerProxier) lookupNodeArchAndOS(ctx context.Context, restClient client.Client, nodeName string) (platform, os string, err error) {
	node := corev1.Node{}
	if err = restClient.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		return "", "", fmt.Errorf("failed to lookup node %s: %w", nodeName, err)
	}
	return node.Status.NodeInfo.Architecture, node.Status.NodeInfo.OperatingSystem, nil
}

func (p *PodContainerProxier) updateContainers(ctx context.Context, containers []corev1.Container, platform, os, kind string) ([]corev1.Container, bool, error) {
	containersReplacement := make([]corev1.Container, 0, len(containers))
	updated := false
	for i := range containers {
		container := containers[i]
		imageRef, err := p.rewriteImage(ctx, container.Image, platform, os)
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

func (p *PodContainerProxier) rewriteImage(ctx context.Context, imageRef, platform, os string) (string, error) {
	for _, transformer := range p.Transformers {
		updatedRef, err := transformer.RewriteImage(imageRef)
		if err != nil {
			return "", fmt.Errorf("transformer %q failed to update imageRef %q: %w", transformer.Name(), imageRef, err)
		}
		if updatedRef != imageRef {
			if found, err := transformer.CheckUpstream(ctx, updatedRef, &v1.Platform{Architecture: platform, OS: os}); err != nil {
				logger.Info(fmt.Sprintf("skipping rewriting %q to %q, could not fetch image manifest: %s", imageRef, updatedRef, err.Error()))
				continue
			} else if !found {
				logger.Info(fmt.Sprintf("skipping rewriting %q to %q, registry reported image not found.", imageRef, updatedRef))
				continue
			}
			return updatedRef, nil
		}
	}
	return imageRef, nil
}

// PodContainerProxier implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (p *PodContainerProxier) InjectDecoder(d *admission.Decoder) error {
	p.Decoder = d
	return nil
}
