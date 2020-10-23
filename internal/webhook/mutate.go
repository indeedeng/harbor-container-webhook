package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/webhook-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io

var logger = ctrl.Log.WithName("mutator")

// ContainerTransformer rewrites docker image references for harbor proxy cache projects.
type ContainerTransformer interface {
	// RewriteImage takes a docker image reference and returns the same image reference rewritten for a harbor
	// proxy cache project endpoint, if one is available, else returns the original image reference.
	RewriteImage(imageRef string) (string, error)
	// Ready returns nil if the transformer is ready to do work.
	Ready() error
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

	initContainers, updatedInit, err := p.updateContainers(pod.Spec.InitContainers)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	containers, updated, err := p.updateContainers(pod.Spec.Containers)
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
		logger.Info(fmt.Sprintf("rewritten pod spec: %#v", pod.Spec))
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (p *PodContainerProxier) updateContainers(containers []corev1.Container) ([]corev1.Container, bool, error) {
	containersReplacement := make([]corev1.Container, 0, len(containers))
	updated := false
	for i := range containers {
		container := containers[i]
		imageRef, err := p.Transformer.RewriteImage(container.Image)
		if err != nil {
			return []corev1.Container{}, false, err
		}
		if !updated {
			updated = imageRef != container.Image
		}
		if imageRef != container.Image {
			logger.Info(fmt.Sprintf("rewriting the image of %q from %q to %q", container.Name, container.Image, imageRef))
		}
		container.Image = imageRef
		containersReplacement = append(containersReplacement, container)
	}
	return containersReplacement, updated, nil
}

// podContainerProxier implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (p *PodContainerProxier) InjectDecoder(d *admission.Decoder) error {
	p.Decoder = d
	return nil
}
