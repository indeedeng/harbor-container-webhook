package dynamic

import (
	"crypto/tls"
	"errors"
	"net/url"
	"os"

	"github.com/hashicorp/go-cleanhttp"

	"indeed.com/devops-incubation/harbor-container-webhook/internal/config"
	"indeed.com/devops-incubation/harbor-container-webhook/internal/webhook"
)

type dynamicTransformer struct {
	cache          ProjectsCache
	harborEndpoint string
}

var _ webhook.ContainerTransformer = (*dynamicTransformer)(nil)

func NewTransformer(conf config.DynamicProxy) webhook.ContainerTransformer {
	harborUser := os.Getenv("HARBOR_USER")
	harborPass := os.Getenv("HARBOR_PASS")
	client := cleanhttp.DefaultClient()
	if conf.SkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		client.Transport = transport
	}
	client.Timeout = conf.Timeout

	projectsCache := NewProjectsCache(client, conf.HarborEndpoint, harborUser, harborPass, conf.ResyncInterval)

	return &dynamicTransformer{
		cache:          projectsCache,
		harborEndpoint: conf.HarborEndpoint,
	}
}

func (d *dynamicTransformer) Ready() error {
	if d.cache.Ready() {
		return nil
	}
	return errors.New("harbor projects cache is unready")
}

func (d *dynamicTransformer) RewriteImage(imageRef string) (string, error) {
	if err := d.Ready(); err != nil {
		return "", err
	}

	projects := d.cache.List()
	harborURL, err := url.Parse(d.harborEndpoint)
	if err != nil {
		return "", err
	}
	proxyMap := registriesToHarborProxies(harborURL.Host, projects)
	registry, err := webhook.RegistryFromImageRef(imageRef)
	if err != nil {
		return "", err
	}

	if rewrite, ok := proxyMap[registry]; ok {
		return webhook.ReplaceRegistryInImageRef(imageRef, rewrite)
	}
	return imageRef, nil
}

// registriesToHarborProxies maps all harbor projects which are configured for a proxy-cache to
// the registry endpoint they are a proxy cache for.
func registriesToHarborProxies(harborEndpoint string, projects []projectWithSummary) map[string]string {
	proxyMap := make(map[string]string)
	for _, project := range projects {
		if project.Registry != nil && project.Registry.URL != "" {
			url, err := url.Parse(project.Registry.URL)
			if err != nil {
				logger.Error(err, "failed to parse url: "+project.Registry.URL)
				continue
			}
			if _, ok := proxyMap[url.Host]; !ok {
				proxyMap[url.Host] = harborEndpoint + "/" + project.Name
			}
		}
	}
	return proxyMap
}
