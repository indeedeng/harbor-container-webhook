package mutate

import (
	"context"
	"encoding/json"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io

// PodContainerProxier mutates init containers and containers to redirect them to the harbor proxy cache if one exists.
type PodContainerProxier struct {
	Client         client.Client
	Decoder        *admission.Decoder
	Cache          ProjectsCache
	HarborEndpoint string
}

func (p *PodContainerProxier) Start(i <-chan struct{}) error {
	return nil
}

// Handle mutates init containers and containers
func (p *PodContainerProxier) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := p.Decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	projects, err := p.Cache.List()
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	proxyMap := registryProxies(p.HarborEndpoint, projects)

	initContainers, err := p.updateContainers(pod.Spec.InitContainers, proxyMap)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	containers, err := p.updateContainers(pod.Spec.Containers, proxyMap)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	pod.Spec.InitContainers = initContainers
	pod.Spec.Containers = containers

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (p *PodContainerProxier) updateContainers(containers []corev1.Container, proxyMap map[string]string) ([]corev1.Container, error) {
	containersReplacement := make([]corev1.Container, 0, len(containers))
	for i := range containers {
		container := containers[i]
		imageRef, err := p.rewriteImage(container.Image, proxyMap)
		if err != nil {
			return []corev1.Container{}, err
		}
		container.Image = imageRef
		containersReplacement = append(containersReplacement, container)
	}
	return containersReplacement, nil
}

func (p *PodContainerProxier) rewriteImage(imageRef string, proxyMap map[string]string) (string, error) {
	registry, err := RegistryFromImageRef(imageRef)
	if err != nil {
		return "", err
	}

	if rewrite, ok := proxyMap[registry]; ok {
		return ReplaceRegistryInImageRef(imageRef, rewrite)
	}
	return "", nil
}

// podContainerProxier implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (p *PodContainerProxier) InjectDecoder(d *admission.Decoder) error {
	p.Decoder = d
	logger.Info("decoder injected: ", "decoder", d)
	return nil
}
