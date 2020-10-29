package static

import (
	"testing"

	"github.com/stretchr/testify/require"

	"indeed.com/devops-incubation/harbor-container-webhook/internal/webhook"
)

func TestStaticTransformer_Ready(t *testing.T) {
	transformer := &staticTransformer{}
	require.Nil(t, transformer.Ready())
}

func TestStaticTransformer_RewriteImage(t *testing.T) {
	transformer := &staticTransformer{
		proxyMap: map[string]string{webhook.BareRegistry: "harbor.example.com/dockerhub-proxy"},
	}

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
			description: "an image from dockerhub should be rewritten",
			image:       "docker.io/library/ubuntu:latest",
			expected:    "harbor.example.com/dockerhub-proxy/library/ubuntu:latest",
		},
		{
			description: "an image from the std library should be rewritten",
			image:       "ubuntu",
			expected:    "harbor.example.com/dockerhub-proxy/library/ubuntu:latest",
		},
	}
	for _, testcase := range tests {
		rewritten, err := transformer.RewriteImage(testcase.image)
		require.NoError(t, err, testcase.description)
		require.Equal(t, testcase.expected, rewritten, testcase.description)
	}
}
