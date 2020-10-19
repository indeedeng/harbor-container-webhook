package static

import (
	"indeed.com/devops-incubation/harbor-container-webhook/internal/config"
	"indeed.com/devops-incubation/harbor-container-webhook/internal/webhook"
)

type staticTransformer struct {
	proxyMap map[string]string
}

func (s *staticTransformer) RewriteImage(imageRef string) (string, error) {
	registry, err := webhook.RegistryFromImageRef(imageRef)
	if err != nil {
		return "", err
	}

	if rewrite, ok := s.proxyMap[registry]; ok {
		return webhook.ReplaceRegistryInImageRef(imageRef, rewrite)
	}
	return imageRef, nil
}

var _ webhook.ContainerTransformer = (*staticTransformer)(nil)

func NewTransformer(conf config.StaticProxy) webhook.ContainerTransformer {
	return &staticTransformer{
		proxyMap: conf.RegistryCaches,
	}
}
