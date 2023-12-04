package webhook

import (
	"testing"

	"github.com/indeedeng-alpha/harbor-container-webhook/internal/config"

	"github.com/stretchr/testify/require"
)

func TestRuleTransformer_RewriteImage(t *testing.T) {
	transformer, err := newRuleTransformer(config.ProxyRule{
		Name:     "test rules",
		Matches:  []string{"^docker.io"},
		Excludes: []string{"^docker.io/(library/)?ubuntu:.*$"},
		Replace:  "harbor.example.com/dockerhub-proxy",
	})
	require.NoError(t, err)

	type testcase struct {
		name     string
		image    string
		platform string
		os       string
		expected string
	}
	tests := []testcase{
		{
			name:     "an image from quay should not be rewritten",
			image:    "quay.io/bitnami/sealed-secrets-controller:latest",
			os:       "linux",
			platform: "amd64",
			expected: "quay.io/bitnami/sealed-secrets-controller:latest",
		},
		{
			name:     "an image from quay without a tag should not be rewritten",
			image:    "quay.io/bitnami/sealed-secrets-controller",
			os:       "linux",
			platform: "amd64",
			expected: "quay.io/bitnami/sealed-secrets-controller",
		},
		{
			name:     "an image from dockerhub explicitly excluded should not be rewritten",
			image:    "docker.io/library/ubuntu:latest",
			os:       "linux",
			platform: "amd64",
			expected: "docker.io/library/ubuntu:latest",
		},
		{
			name:     "a bare image from dockerhub explicitly excluded should not be rewritten",
			image:    "ubuntu",
			os:       "linux",
			platform: "amd64",
			expected: "ubuntu",
		},
		{
			name:     "an image from dockerhub should be rewritten",
			image:    "docker.io/library/centos:latest",
			os:       "linux",
			platform: "amd64",
			expected: "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			name:     "an image from the std library should be rewritten",
			image:    "centos",
			os:       "linux",
			platform: "amd64",
			expected: "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rewritten, err := transformer.RewriteImage(tc.image)
			require.NoError(t, err)
			require.Equal(t, tc.expected, rewritten)
		})
	}
}
