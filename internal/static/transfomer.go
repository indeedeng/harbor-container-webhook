package static

import (
	"errors"
	"net/http"
	"time"

	"indeed.com/devops-incubation/harbor-container-webhook/internal/config"
	"indeed.com/devops-incubation/harbor-container-webhook/internal/webhook"
)

type staticTransformer struct {
	proxyMap       map[string]string
	harborEndpoint string
	HarborVerifier func(string) (bool, error)
}

func VerifyHarborIsRunning(endpoint string) (bool, error) {
	timeout := time.Second
	client := http.Client{
		Timeout: timeout,
	}
	res, err := client.Get(endpoint + "/api/version")
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return false, err
	}
	if res.StatusCode != 200 {
		return false, errors.New("harbor API server did not return 200 status code")
	}
	return true, nil
}

func (s *staticTransformer) RewriteImage(imageRef string) (string, error) {
	registry, err := webhook.RegistryFromImageRef(imageRef)
	if err != nil {
		return "", err
	}

	running, err := s.HarborVerifier(s.harborEndpoint)
	if running {
		if rewrite, ok := s.proxyMap[registry]; ok {
			return webhook.ReplaceRegistryInImageRef(imageRef, rewrite)
		}
	}

	return imageRef, err
}

func (s *staticTransformer) Ready() error {
	return nil
}

var _ webhook.ContainerTransformer = (*staticTransformer)(nil)

func NewTransformer(conf config.StaticProxy) webhook.ContainerTransformer {
	return &staticTransformer{
		proxyMap:       conf.RegistryCaches,
		harborEndpoint: conf.HarborEndpoint,
		HarborVerifier: VerifyHarborIsRunning,
	}
}
