package webhook

import (
	"context"
	"testing"

	"github.com/indeedeng-alpha/harbor-container-webhook/internal/config"

	"github.com/stretchr/testify/require"
)

func TestPodContainerProxier_rewriteImage(t *testing.T) {
	transformers, err := MakeTransformers([]config.ProxyRule{
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
	}, nil)
	require.NoError(t, err)
	proxier := PodContainerProxier{
		Transformers: transformers,
	}

	type testcase struct {
		name     string
		image    string
		platform string
		os       string
		expected string
	}
	tests := []testcase{
		{
			name:     "an image from quay should be rewritten",
			image:    "quay.io/bitnami/sealed-secrets-controller:latest",
			os:       "linux",
			platform: "amd64",
			expected: "harbor.example.com/quay-proxy/bitnami/sealed-secrets-controller:latest",
		},
		{
			name:     "an image from quay without a tag should be rewritten",
			image:    "quay.io/bitnami/sealed-secrets-controller",
			os:       "linux",
			platform: "amd64",
			expected: "harbor.example.com/quay-proxy/bitnami/sealed-secrets-controller:latest",
		},
		{
			name:     "an image from docker.io with ubuntu should be rewritten to the ubuntu proxy",
			image:    "docker.io/library/ubuntu:latest",
			os:       "linux",
			platform: "amd64",
			expected: "harbor.example.com/ubuntu-proxy/library/ubuntu:latest",
		},
		{
			name:     "a bare ubuntu image from docker.io should be rewritten to the ubuntu proxy",
			image:    "ubuntu",
			os:       "linux",
			platform: "amd64",
			expected: "harbor.example.com/ubuntu-proxy/library/ubuntu:latest",
		},
		{
			name:     "an image from docker.io should be rewritten",
			image:    "docker.io/library/centos:latest",
			os:       "linux",
			platform: "amd64",
			expected: "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			name:     "a bare image from docker.io should be rewritten",
			image:    "centos",
			os:       "linux",
			platform: "amd64",
			expected: "harbor.example.com/dockerhub-proxy/library/centos:latest",
		},
		{
			name:     "an image from gcr should not be rewritten",
			image:    "k8s.gcr.io/kubernetes",
			os:       "linux",
			platform: "amd64",
			expected: "k8s.gcr.io/kubernetes",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rewritten, err := proxier.rewriteImage(context.TODO(), tc.image)
			require.NoError(t, err)
			require.Equal(t, tc.expected, rewritten)
		})
	}
}
