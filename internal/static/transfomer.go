package static

import (
	"crypto/tls"
	"errors"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"

	"indeed.com/devops-incubation/harbor-container-webhook/internal/config"
	"indeed.com/devops-incubation/harbor-container-webhook/internal/webhook"
)

type staticTransformer struct {
	proxyMap       map[string]string
	harborEndpoint string
	HarborVerifier func(string) (bool, error)
}

type harborCheck struct {
	client *http.Client
}

func (h *harborCheck) verifyHarborIsRunning(endpoint string) (bool, error) {
	res, err := h.client.Get(endpoint + "/api/version")
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
	// TODO: (cnmcavoy) move http client setup to shared logic with the dynamic transformer?
	client := cleanhttp.DefaultClient()
	if conf.SkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		client.Transport = transport
	}
	client.Timeout = conf.Timeout
	harborCheck := &harborCheck{client: client}
	harborVerifier := harborCheck.verifyHarborIsRunning
	if !conf.VerifyHarborAPI {
		harborVerifier = func(string) (bool, error) {
			return true, nil
		}
	}

	return &staticTransformer{
		proxyMap:       conf.RegistryCaches,
		harborEndpoint: conf.HarborEndpoint,
		HarborVerifier: harborVerifier,
	}
}
