package static

import (
	"strings"

	"indeed.com/devops-incubation/harbor-container-webhook/internal/config"
	"indeed.com/devops-incubation/harbor-container-webhook/internal/mutate"
)

type staticTransformer struct {
	proxyMap map[string]string
}

func (s *staticTransformer) RewriteImage(imageRef string) (string, error) {
	registry, err := mutate.RegistryFromImageRef(imageRef)
	if err != nil {
		return "", err
	}

	if rewrite, ok := s.proxyMap[registry]; ok {
		return mutate.ReplaceRegistryInImageRef(imageRef, rewrite)
	}
	return imageRef, nil
}

var _ mutate.ContainerTransformer = (*staticTransformer)(nil)

func NewTransformer(conf config.StaticProxy) mutate.ContainerTransformer {
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
