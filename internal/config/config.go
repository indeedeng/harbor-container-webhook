package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

func LoadConfiguration(path string) (*Configuration, error) {
	conf := &Configuration{}
	data, err := os.ReadFile(path)
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
	CertDir string `yaml:"certDir"`
	// MetricsAddr is the address the metric endpoint binds to.
	MetricsAddr string `yaml:"metricsAddr"`
	// HealthAddr is the address the readiness and health probes are mounted to.
	HealthAddr string `yaml:"healthAddr"`
	// Rules is the list of directives to use to evaluate pod container images.
	Rules []ProxyRule `yaml:"rules"`
	// Verbose enables trace logging.
	Verbose bool `yaml:"verbose"`
}

// ProxyRule contains a list of regex rules used to match against images. Image references that match and are not
// excluded have their registry rewritten with the replacement string.
type ProxyRule struct {
	// Name of the ProxyRule.
	Name string `yaml:"name"`
	// Matches is a list of regular expressions that match a registry in an image, e.g '^docker.io'.
	Matches []string `yaml:"matches"`
	// Excludes is a list of regular expressions whose images that match should be excluded from this rule.
	Excludes []string `yaml:"excludes"`
	// Replace is the string used to rewrite the registry in matching rules.
	Replace string `yaml:"replace"`
	// CheckUpstream enables an additional check to ensure the image manifest exists before rewriting.
	// If the webhook lacks permissions to fetch the image manifest or the registry is down, the image
	// will not be rewritten. Experimental.
	CheckUpstream bool `yaml:"checkUpstream"`
}
