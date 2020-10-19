package static

import (
	"strings"

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
	proxyMap := make(map[string]string, len(conf.RegistryCaches))
	for _, cache := range conf.RegistryCaches {
		s := strings.Split(cache, ":")
		if len(s) != 2 {
			panic("unexpected number of ':' separator in static transformer registry caches: " + cache)
		}
		registry := s[0]
		proxyCache := s[1]
		proxyMap[registry] = proxyCache
	}
	return &staticTransformer{
		proxyMap: proxyMap,
	}
}
