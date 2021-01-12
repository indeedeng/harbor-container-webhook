package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

func LoadConfiguration(path string) (*Configuration, error) {
	conf := &Configuration{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

// Configuration loads and keeps the related configuration items for the webhook.
type Configuration struct {
	// Port that the webhook listens on for admission review submissions
	Port int `yaml:"port"`
	// CertDir the directory that contains the webhook server key and certificate.
	// If not set, webhook server would look up the server key and certificate in
	// {TempDir}/k8s-webhook-server/serving-certs. The server key and certificate
	// must be named tls.key and tls.crt, respectively.
	CertDir string `yaml:"cert_dir"`
	// MetricsAddr is the address the metric endpoint binds to.
	MetricsAddr string `yaml:"metrics_addr"`
	// HealthAddr is the address the readiness and health probes are mounted to.
	HealthAddr string        `yaml:"health_addr"`
	Dynamic    *DynamicProxy `yaml:"dynamic"`
	Static     *StaticProxy  `yaml:"static"`
	Verbose    bool          `yaml:"verbose"`
}

// DynamicProxy queries the Harbor API to discover projects, and find projects in harbor with a proxy cache
// endpoint configured. For each such project, it inspects pod container images, and rewrites container images for
// any container to pull from the proxy cache instead. DynamicProxy requires API access and harbor credentials set
// as HARBOR_USER and HARBOR_PASS in order to query the Harbor API.
type DynamicProxy struct {
	// ResyncInterval configures how often projects & proxy cache registry info is refreshed from the harbor API.
	ResyncInterval time.Duration `yaml:"resync_interval"`
	// Timeout sets the http.Client Timeout for harbor API requests.
	Timeout time.Duration `yaml:"timeout"`
	// SkipTLSVerify if set configures the http.Client to not validate the harbor API certificate for requests.
	SkipTLSVerify bool `yaml:"skip_tls_verify"`
	// HarborEndpoint is the address to query for harbor projects and discover proxy cache configuration.
	HarborEndpoint string `yaml:"harbor_endpoint"`
}

// StaticProxy configures a static transformer for pods and rewrites container image in a sed-like fashion.
// For every pod, it inspects container images, and rewrites the container images according to the supplied
// configuration below. The advantage of the static proxy configuration is that no auth or API access to harbor
// is necessary. However, if the harbor project is renamed or deleted for the proxy cache, the static proxy could
// damage the cluster by configuring containers incorrectly.
type StaticProxy struct {
	// RegistryCaches is an map of registries to harbor projects
	RegistryCaches map[string]string `yaml:"registry_caches"`
	// HarborEndpoint is the address to query for harbor projects and discover proxy cache configuration.
	HarborEndpoint string `yaml:"harbor_endpoint"`
}
