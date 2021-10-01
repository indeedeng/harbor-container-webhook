package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"

	"indeed.com/devops-incubation/harbor-container-webhook/internal/config"
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
		expected    string
	}
	tests := []testcase{
		{
			description: "an image from quay should be rewritten",
			image:       "quay.io/bitnami/sealed-secrets-controller:latest",
			expected:    "harbor.example.com/quay-proxy/bitnami/sealed-secrets-controller:latest",
		},
		{
			description: "an image from quay without a tag should be rewritten",
			image:       "quay.io/bitnami/sealed-secrets-controller",
			expected:    "harbor.example.com/quay-proxy/bitnami/sealed-secrets-controller:latest",
		},
		{
			description: "an image from docker.io with ubuntu should be rewritten to the ubuntu proxy",
			image:       "docker.io/library/ubuntu:latest",
			expected:    "harbor.example.com/ubuntu-proxy/library/ubuntu:latest",
		},
		{
			description: "a bare ubuntu image from docker.io should be rewritten to the ubuntu proxy",
			image:       "ubuntu",
			expected:    "harbor.example.com/ubuntu-proxy/library/ubuntu:latest",
		},
		{
			description: "an image from docker.io should be rewritten",
			image:       "docker.io/library/centos:latest",
			expected:    "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			description: "a bare image from docker.io should be rewritten",
			image:       "centos",
			expected:    "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			description: "an image from gcr should not be rewritten",
			image:       "k8s.gcr.io/kubernetes",
			expected:    "k8s.gcr.io/kubernetes",
		},
	}
	for _, testcase := range tests {
		rewritten, err := transformer.RewriteImage(testcase.image)
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
		expected    string
	}
	tests := []testcase{
		{
			description: "an image from quay should not be rewritten",
			image:       "quay.io/bitnami/sealed-secrets-controller:latest",
			expected:    "quay.io/bitnami/sealed-secrets-controller:latest",
		},
		{
			description: "an image from quay without a tag should not be rewritten",
			image:       "quay.io/bitnami/sealed-secrets-controller",
			expected:    "quay.io/bitnami/sealed-secrets-controller",
		},
		{
			description: "an image from dockerhub explicitly excluded should not be rewritten",
			image:       "docker.io/library/ubuntu:latest",
			expected:    "docker.io/library/ubuntu:latest",
		},
		{
			description: "a bare image from dockerhub explicitly excluded should not be rewritten",
			image:       "ubuntu",
			expected:    "ubuntu",
		},
		{
			description: "an image from dockerhub should be rewritten",
			image:       "docker.io/library/centos:latest",
			expected:    "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			description: "an image from the std library should be rewritten",
			image:       "centos",
			expected:    "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
	}
	for _, testcase := range tests {
		rewritten, err := transformer.RewriteImage(testcase.image)
		require.NoError(t, err, testcase.description)
		require.Equal(t, testcase.expected, rewritten, testcase.description)
	}
}
