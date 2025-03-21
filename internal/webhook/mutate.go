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

var (
	logger = ctrl.Log.WithName("mutator")
)

// PodContainerProxier mutates init containers and containers to redirect them to the harbor proxy cache if one exists.
type PodContainerProxier struct {
	Client       client.Client
	Decoder      admission.Decoder
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

	// container images
	initContainers, updatedInit, err := p.updateContainers(ctx, pod.Spec.InitContainers, "init")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	containers, updated, err := p.updateContainers(ctx, pod.Spec.Containers, "normal")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	pod.Spec.InitContainers = initContainers
	pod.Spec.Containers = containers

	if !updated && !updatedInit {
		return admission.Allowed("no updates")
	}

	// imagePullSecrets
	imagePullSecrets, _, err := p.updateImagePullSecrets(pod.Name, pod.Spec.ImagePullSecrets)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	pod.Spec.ImagePullSecrets = imagePullSecrets

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (p *PodContainerProxier) lookupNodeArchAndOS(ctx context.Context, restClient client.Client, nodeName string) (platform, os string, err error) {
	node := corev1.Node{}
	if err = restClient.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		return "", "", fmt.Errorf("failed to lookup node %s: %w", nodeName, err)
	}
	logger.Info(fmt.Sprintf("node %v", node))
	return node.Status.NodeInfo.Architecture, node.Status.NodeInfo.OperatingSystem, nil
}

func (p *PodContainerProxier) updateContainers(ctx context.Context, containers []corev1.Container, _ string) ([]corev1.Container, bool, error) {
	containersReplacement := make([]corev1.Container, 0, len(containers))
	updated := false
	for i := range containers {
		container := containers[i]
		imageRef, err := p.rewriteImage(ctx, container.Image)
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

func (p *PodContainerProxier) rewriteImage(ctx context.Context, imageRef string) (string, error) {
	for _, transformer := range p.Transformers {
		updatedRef, err := transformer.RewriteImage(imageRef)
		if err != nil {
			return "", fmt.Errorf("transformer %q failed to update imageRef %q: %w", transformer.Name(), imageRef, err)
		}
		if updatedRef != imageRef {
			if found, err := transformer.CheckUpstream(ctx, updatedRef); err != nil {
				logger.Info(fmt.Sprintf("transformer %q skipping rewriting %q to %q, could not fetch image manifest: %s", transformer.Name(), imageRef, updatedRef, err.Error()))
				continue
			} else if !found {
				logger.Info(fmt.Sprintf("transformer %q skipping rewriting %q to %q, registry reported image not found.", transformer.Name(), imageRef, updatedRef))
				continue
			}
			logger.Info(fmt.Sprintf("transformer %q rewriting %q to %q", transformer.Name(), imageRef, updatedRef))
			return updatedRef, nil
		}
	}
	return imageRef, nil
}

// PodContainerProxier implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (p *PodContainerProxier) InjectDecoder(d admission.Decoder) error {
	p.Decoder = d
	return nil
}

func (p *PodContainerProxier) updateImagePullSecrets(podName string, imagePullSecrets []corev1.LocalObjectReference) (newImagePullSecrets []corev1.LocalObjectReference, updated bool, err error) {
	for _, transformer := range p.Transformers {
		updated, newImagePullSecrets, err = transformer.RewriteImagePullSecrets(imagePullSecrets)
		if err != nil {
			return imagePullSecrets, false, err
		}
		if !updated {
			return imagePullSecrets, false, nil
		}
		logger.Info(fmt.Sprintf("rewriting the imagePullSecrets of the pod %s from %q to %q", podName, imagePullSecrets, newImagePullSecrets))
	}
	return newImagePullSecrets, updated, nil
}
