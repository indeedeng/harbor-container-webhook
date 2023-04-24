package webhook

import (
	"testing"

	"github.com/indeedeng-alpha/harbor-container-webhook/internal/config"

	"github.com/stretchr/testify/require"
)

func TestMultiTransformer_RewriteImage(t *testing.T) {
	transformer, err := NewMultiTransformer([]config.ProxyRule{
		{
			Name:     "docker.io proxy cache except ubuntu",
			Matches:  []string{"^docker.io"},
			Excludes: []string{"^docker.io/(library/)?ubuntu:.*$"},
			Replace:  "harbor.example.com/dockerhub-proxy",
		},
		{
			Name:    "quay.io proxy cache",
			Matches: []string{"^quay.io"},
			Replace: "harbor.example.com/quay-proxy",
		},
		{
			Name:    "docker.io proxy cache but only ubuntu",
			Matches: []string{"^docker.io/(library/)?ubuntu"},
			Replace: "harbor.example.com/ubuntu-proxy",
		},
	})
	require.NoError(t, err)

	type testcase struct {
		description string
		image       string
		platform    string
		os          string
		expected    string
	}
	tests := []testcase{
		{
			description: "an image from quay should be rewritten",
			image:       "quay.io/bitnami/sealed-secrets-controller:latest",
			os:          "linux",
			platform:    "amd64",
			expected:    "harbor.example.com/quay-proxy/bitnami/sealed-secrets-controller:latest",
		},
		{
			description: "an image from quay without a tag should be rewritten",
			image:       "quay.io/bitnami/sealed-secrets-controller",
			os:          "linux",
			platform:    "amd64",
			expected:    "harbor.example.com/quay-proxy/bitnami/sealed-secrets-controller:latest",
		},
		{
			description: "an image from docker.io with ubuntu should be rewritten to the ubuntu proxy",
			image:       "docker.io/library/ubuntu:latest",
			os:          "linux",
			platform:    "amd64",
			expected:    "harbor.example.com/ubuntu-proxy/library/ubuntu:latest",
		},
		{
			description: "a bare ubuntu image from docker.io should be rewritten to the ubuntu proxy",
			image:       "ubuntu",
			os:          "linux",
			platform:    "amd64",
			expected:    "harbor.example.com/ubuntu-proxy/library/ubuntu:latest",
		},
		{
			description: "an image from docker.io should be rewritten",
			image:       "docker.io/library/centos:latest",
			os:          "linux",
			platform:    "amd64",
			expected:    "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			description: "a bare image from docker.io should be rewritten",
			image:       "centos",
			os:          "linux",
			platform:    "amd64",
			expected:    "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			description: "an image from gcr should not be rewritten",
			image:       "k8s.gcr.io/kubernetes",
			os:          "linux",
			platform:    "amd64",
			expected:    "k8s.gcr.io/kubernetes",
		},
	}
	for _, testcase := range tests {
		rewritten, err := transformer.RewriteImage(testcase.image, testcase.platform, testcase.os)
		require.NoError(t, err, testcase.description)
		require.Equal(t, testcase.expected, rewritten, testcase.description)
	}
}

func TestRuleTransformer_RewriteImage(t *testing.T) {
	transformer, err := newRuleTransformer(config.ProxyRule{
		Name:     "test rules",
		Matches:  []string{"^docker.io"},
		Excludes: []string{"^docker.io/(library/)?ubuntu:.*$"},
		Replace:  "harbor.example.com/dockerhub-proxy",
	})
	require.NoError(t, err)

	type testcase struct {
		description string
		image       string
		platform    string
		os          string
		expected    string
	}
	tests := []testcase{
		{
			description: "an image from quay should not be rewritten",
			image:       "quay.io/bitnami/sealed-secrets-controller:latest",
			os:          "linux",
			platform:    "amd64",
			expected:    "quay.io/bitnami/sealed-secrets-controller:latest",
		},
		{
			description: "an image from quay without a tag should not be rewritten",
			image:       "quay.io/bitnami/sealed-secrets-controller",
			os:          "linux",
			platform:    "amd64",
			expected:    "quay.io/bitnami/sealed-secrets-controller",
		},
		{
			description: "an image from dockerhub explicitly excluded should not be rewritten",
			image:       "docker.io/library/ubuntu:latest",
			os:          "linux",
			platform:    "amd64",
			expected:    "docker.io/library/ubuntu:latest",
		},
		{
			description: "a bare image from dockerhub explicitly excluded should not be rewritten",
			image:       "ubuntu",
			os:          "linux",
			platform:    "amd64",
			expected:    "ubuntu",
		},
		{
			description: "an image from dockerhub should be rewritten",
			image:       "docker.io/library/centos:latest",
			os:          "linux",
			platform:    "amd64",
			expected:    "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			description: "an image from the std library should be rewritten",
			image:       "centos",
			os:          "linux",
			platform:    "amd64",
			expected:    "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
	}
	for _, testcase := range tests {
		rewritten, err := transformer.RewriteImage(testcase.image, testcase.platform, testcase.os)
		require.NoError(t, err, testcase.description)
		require.Equal(t, testcase.expected, rewritten, testcase.description)
	}
}
