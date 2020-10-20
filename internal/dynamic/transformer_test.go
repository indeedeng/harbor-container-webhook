package dynamic

import (
	"testing"

	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/replication/model"

	"github.com/stretchr/testify/require"

	"indeed.com/devops-incubation/harbor-container-webhook/internal/webhook"
)

func TestDynamicTransformer_Ready_Unready(t *testing.T) {
	mockProjectsCache := &mockProjectsCache{}
	defer mockProjectsCache.AssertExpectations(t)
	transformer := &dynamicTransformer{
		cache: mockProjectsCache,
	}

	mockProjectsCache.On("Ready").Return(false).Once()

	require.Error(t, transformer.Ready())
}

func TestDynamicTransformer_Ready(t *testing.T) {
	mockProjectsCache := &mockProjectsCache{}
	defer mockProjectsCache.AssertExpectations(t)
	transformer := &dynamicTransformer{
		cache: mockProjectsCache,
	}

	mockProjectsCache.On("Ready").Return(true).Once()

	require.Nil(t, transformer.Ready())
}

func TestDynamicTransformer_RewriteImage(t *testing.T) {
	mockProjectsCache := &mockProjectsCache{}
	defer mockProjectsCache.AssertExpectations(t)
	transformer := &dynamicTransformer{
		cache:          mockProjectsCache,
		harborEndpoint: "https://harbor.example.com",
	}

	projects := []projectWithSummary{
		{
			Project: models.Project{
				Name: "foo",
			},
		},
		{
			Project: models.Project{
				Name: "bar",
			},
		},
		{
			Project: models.Project{
				Name: "dockerhub-proxy",
			},
			ProjectSummary: models.ProjectSummary{
				Registry: &model.Registry{
					Name: "dockerhub",
					URL:  "https://registry.hub.docker.com",
				},
			},
		},
		{
			Project: models.Project{
				Name: "quz-proxy",
			},
			ProjectSummary: models.ProjectSummary{
				Registry: &model.Registry{
					Name: "quz",
					URL:  "https://quz.example.com",
				},
			},
		},
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
			image:       "registry.hub.docker.com/library/ubuntu:latest",
			expected:    "harbor.example.com/dockerhub-proxy/library/ubuntu:latest",
		},
		{
			description: "an image from the std library should be rewritten",
			image:       "ubuntu",
			expected:    "harbor.example.com/dockerhub-proxy/library/ubuntu",
		},
		{
			description: "an image from quz.example.com should be rewritten",
			image:       "quz.example.com/example/quz",
			expected:    "harbor.example.com/quz-proxy/example/quz",
		},
	}

	mockProjectsCache.On("Ready").Return(true).Times(len(tests))
	mockProjectsCache.On("List").Return(projects).Times(len(tests))

	for _, testcase := range tests {
		rewritten, err := transformer.RewriteImage(testcase.image)
		require.NoError(t, err, testcase.description)
		require.Equal(t, testcase.expected, rewritten, testcase.description)
	}

}

func Test_registriesToHarborProxies(t *testing.T) {
	type testcase struct {
		description     string
		projects        []projectWithSummary
		expectedProxies map[string]string
	}
	testcases := []testcase{
		{
			description:     "no op",
			projects:        []projectWithSummary{},
			expectedProxies: map[string]string{},
		},
		{
			description: "map bare dockerhub to harbor project",
			projects: []projectWithSummary{
				{
					Project: models.Project{
						Name: "proxy-cache",
					},
					ProjectSummary: models.ProjectSummary{
						Registry: &model.Registry{
							URL: "https://" + webhook.BareRegistry,
						},
					},
				},
			},
			expectedProxies: map[string]string{
				webhook.BareRegistry: "harbor.example.com/proxy-cache",
			},
		},
	}
	for _, testcase := range testcases {
		proxyMap := registriesToHarborProxies("harbor.example.com", testcase.projects)
		require.Equal(t, testcase.expectedProxies, proxyMap, testcase.description)
	}
}
