package webhook

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

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
		{
			Name:    "docker.io proxy cache with imagePullSecret change",
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

func TestPodContainerProxier_updateImagePullSecretsWithReplaceEnabled(t *testing.T) {
	transformers, err := MakeTransformers([]config.ProxyRule{
		{
			Name:                    "docker.io proxy cache with imagePullSecrets change",
			Matches:                 []string{"^docker.io"},
			Replace:                 "harbor.example.com/dockerhub-proxy",
			ReplaceImagePullSecrets: true,
			AuthSecretName:          "secret-test",
		},
	}, nil)
	require.NoError(t, err)
	proxier := PodContainerProxier{
		Transformers: transformers,
	}

	type testcase struct {
		name             string
		imagePullSecrets []corev1.LocalObjectReference
		platform         string
		os               string
		expected         []corev1.LocalObjectReference
	}
	tests := []testcase{
		{
			name:             "imagePullSecrets is empty, replacement is expected and secret name should be added",
			imagePullSecrets: []corev1.LocalObjectReference{},
			os:               "linux",
			platform:         "amd64",
			expected:         []corev1.LocalObjectReference{{Name: "secret-test"}},
		},
		{
			name:             "imagePullSecrets has a secret, replacement is expected and secret name should be added",
			imagePullSecrets: []corev1.LocalObjectReference{{Name: "mysecret"}},
			os:               "linux",
			platform:         "amd64",
			expected:         []corev1.LocalObjectReference{{Name: "mysecret"}, {Name: "secret-test"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newImagePullSecrets, _, err := proxier.updateImagePullSecrets(tc.imagePullSecrets)
			require.NoError(t, err)
			require.Equal(t, tc.expected, newImagePullSecrets)
		})
	}
}

func TestPodContainerProxier_updateImagePullSecretsWithReplaceDinabled(t *testing.T) {
	transformers, err := MakeTransformers([]config.ProxyRule{
		{
			Name:    "docker.io proxy cache without imagePullSecrets change",
			Matches: []string{"^docker.io"},
			Replace: "harbor.example.com/dockerhub-proxy",
		},
	}, nil)
	require.NoError(t, err)
	proxier := PodContainerProxier{
		Transformers: transformers,
	}

	type testcase struct {
		name             string
		imagePullSecrets []corev1.LocalObjectReference
		platform         string
		os               string
		expected         []corev1.LocalObjectReference
	}
	tests := []testcase{
		{
			name:             "imagePullSecrets is empty, replacement is not expected",
			imagePullSecrets: []corev1.LocalObjectReference{},
			os:               "linux",
			platform:         "amd64",
			expected:         []corev1.LocalObjectReference{},
		},
		{
			name:             "imagePullSecrets has a secret, replacement is not expected",
			imagePullSecrets: []corev1.LocalObjectReference{{Name: "mysecret"}},
			os:               "linux",
			platform:         "amd64",
			expected:         []corev1.LocalObjectReference{{Name: "mysecret"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newImagePullSecrets, _, err := proxier.updateImagePullSecrets(tc.imagePullSecrets)
			require.NoError(t, err)
			require.Equal(t, tc.expected, newImagePullSecrets)
		})
	}
}
