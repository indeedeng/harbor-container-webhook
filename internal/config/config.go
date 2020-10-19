package config

import (
	"io/ioutil"
	"time"

	"sigs.k8s.io/yaml"
)

func LoadConfiguration(path string) (*Configuration, error) {
	conf := &Configuration{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

// Configuration loads and keeps the related configuration items for the webhook.
type Configuration struct {
	Port int `yaml:"port"`
	// the directory that contains the webhook server key and certificate.
	CertDir string `yaml:"cert-dir"`
	// MetricsAddr is the address the metric endpoint binds to.
	MetricsAddr string `yaml:"metrics-addr"`
	// Enabling this will ensure there is only one active controller manager.
	EnableLeaderElection bool         `yaml:"enable-leader-election"`
	Dynamic              DynamicProxy `yaml:"dynamic"`
	Static               StaticProxy  `yaml:"static"`
}

// DynamicProxy queries the Harbor API to discover projects, and find projects in harbor with a proxy cache
// endpoint configured. For each such project, it inspects pod container images, and rewrites container images for
// any container to pull from the proxy cache instead. DynamicProxy requires API access and harbor credentials set
// as HARBOR_USER and HARBOR_PASS in order to query the Harbor API.
type DynamicProxy struct {
	// Enables dynamic lookup of harbor projects to discover proxy cache information.
	Enabled bool `yaml:"enabled"`
	// ResyncInterval configures how often projects & proxy cache registry info is refreshed from the harbor API.
	ResyncInterval time.Duration `yaml:"port"`
	// Timeout sets the http.Client Timeout for harbor API requests.
	Timeout time.Duration `yaml:"timeout"`
	// SkipTLSVerify if set configures the http.Client to not validate the harbor API certificate for requests.
	SkipTLSVerify bool `yaml:"skip-tls-verify"`
	// HarborEndpoint is the address to query for harbor projects and discover proxy cache configuration.
	HarborEndpoint string `yaml:"harbor-endpoint"`
}

// StaticProxy configures a static transformer for pods that mutate the container image in a sed-like fashion.
// For every pod, it inspects container images, and rewrites the container images according to the supplied
// configuration below. The advantage of the static proxy configuration is that no auth or API access to harbor
// is necessary. However, if the harbor project is renamed or deleted for the proxy cache, the static proxy could
// damage the cluster by configuring containers incorrectly.
type StaticProxy struct {
	// Enables static registry -> harbor project transformer for pods.
	Enabled bool `yaml:"enabled"`
	// RegistryCaches is an array of strings in the form of "registry:harbor-project". For example
	// to configure docker hub images to pull from a harbor project named "dockerhub-cache", you would configure:
	// "registry.hub.docker.com:my-harbor-registry.com/dockerhub-cache"
	RegistryCaches []string `yaml:"registry-caches"`
}
