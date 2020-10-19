package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_RegistryFromImageRef(t *testing.T) {
	type testcase struct {
		description      string
		imageRef         string
		expectedRegistry string
	}
	tests := []testcase{
		{
			description:      "image reference with hostname with port and image tag set",
			imageRef:         "some_host:443/public/busybox:latest",
			expectedRegistry: "some_host:443",
		},
		{
			description:      "image reference with hostname and image tag set",
			imageRef:         "some_host/public/busybox:latest",
			expectedRegistry: "some_host",
		},
		{
			description:      "image reference with hostname with port and no image tag set",
			imageRef:         "some_host:443/public/busybox",
			expectedRegistry: "some_host:443",
		},
		{
			description:      "image reference with hostname and no image tag set",
			imageRef:         "some_host/public/busybox",
			expectedRegistry: "some_host",
		},
		{
			description:      "image reference with hostname with port and image sha set",
			imageRef:         "some_host:443/public/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			expectedRegistry: "some_host:443",
		},
		{
			description:      "image reference with hostname and image sha set",
			imageRef:         "some_host/public/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			expectedRegistry: "some_host",
		},
		{
			description:      "image reference with url and image tag set",
			imageRef:         "example.com/busybox:latest",
			expectedRegistry: "example.com",
		},
		{
			description:      "image reference with url and no image tag set",
			imageRef:         "example.com/busybox",
			expectedRegistry: "example.com",
		},
		{
			description:      "image reference with url, project and no image tag set",
			imageRef:         "example.com/nginxinc/nginx-unprivileged",
			expectedRegistry: "example.com",
		},
		{
			description:      "image reference with url and image sha set",
			imageRef:         "example.com/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			expectedRegistry: "example.com",
		},
		{
			description:      "bare image reference with image tag set",
			imageRef:         "busybox:latest",
			expectedRegistry: BareRegistry,
		},
		{
			description:      "bare image reference with project and no image tag set",
			imageRef:         "nginxinc/nginx-unprivileged",
			expectedRegistry: BareRegistry,
		},
		{
			description:      "bare image reference with and no image tag set",
			imageRef:         "busybox",
			expectedRegistry: BareRegistry,
		},
		{
			description:      "bare image reference with image sha set",
			imageRef:         "busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			expectedRegistry: BareRegistry,
		},
	}
	for _, testcase := range tests {
		output, err := RegistryFromImageRef(testcase.imageRef)
		require.NoError(t, err)
		require.Equal(t, testcase.expectedRegistry, output, testcase.description)
	}
}

func Test_IsLibraryImage(t *testing.T) {
	type testcase struct {
		imageRef string
		expected bool
	}
	tests := []testcase{
		{
			imageRef: "nginxinc/nginx-unprivileged",
			expected: false,
		},
		{
			imageRef: "nginx",
			expected: true,
		},
		{
			imageRef: "nginxinc/nginx-unprivileged:v1.19.0",
			expected: false,
		},
		{
			imageRef: "nginx:v1.19.0",
			expected: true,
		},
		{
			imageRef: "nginxinc/nginx-unprivileged@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			expected: false,
		},
		{
			imageRef: "nginx@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			expected: true,
		},
	}
	for _, test := range tests {
		actual := IsLibraryImage(test.imageRef)
		require.Equal(t, test.expected, actual, test.imageRef)
	}
}

func Test_RegistryFromImageRef_EmptyErr(t *testing.T) {
	_, err := RegistryFromImageRef("")
	require.EqualError(t, err, "image reference `` invalid, unable to parse registry or image name")
}

func Test_ReplaceRegistryInImageRef(t *testing.T) {
	type testcase struct {
		description string
		imageRef    string
		newRegistry string
		expectedRef string
	}
	tests := []testcase{
		{
			description: "image reference with hostname with port and image tag set",
			imageRef:    "some_host:443/public/busybox:latest",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/public/busybox:latest",
		},
		{
			description: "image reference with hostname and image tag set",
			imageRef:    "some_host/public/busybox:latest",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/public/busybox:latest",
		},
		{
			description: "image reference with hostname with port and no image tag set",
			imageRef:    "some_host:443/public/busybox",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/public/busybox",
		},
		{
			description: "image reference with hostname and no image tag set",
			imageRef:    "some_host/public/busybox",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/public/busybox",
		},
		{
			description: "image reference with hostname with port and image sha set",
			imageRef:    "some_host:443/public/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/public/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
		},
		{
			description: "image reference with hostname and image sha set",
			imageRef:    "some_host/public/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/public/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
		},
		{
			description: "image reference with url and image tag set",
			imageRef:    "example.com/busybox:latest",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/busybox:latest",
		},
		{
			description: "image reference with url and no image tag set",
			imageRef:    "example.com/busybox",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/busybox",
		},
		{
			description: "image reference with url and image sha set",
			imageRef:    "example.com/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
		},
		{
			description: "bare image reference with image tag set",
			imageRef:    "busybox:latest",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/busybox:latest",
		},
		{
			description: "bare image reference with and no image tag set",
			imageRef:    "busybox",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/busybox",
		},
		{
			description: "bare image reference with image sha set",
			imageRef:    "busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
			newRegistry: "harbor.example.com/proxy-cache",
			expectedRef: "harbor.example.com/proxy-cache/busybox@sha256:c34ce3c1fcc0c7431e1392cc3abd0dfe2192ffea1898d5250f199d3ac8d87",
		},
	}
	for _, testcase := range tests {
		output, err := ReplaceRegistryInImageRef(testcase.imageRef, testcase.newRegistry)
		require.NoError(t, err)
		require.Equal(t, testcase.expectedRef, output, testcase.description)
	}
}
