package mutate

import (
	"testing"

	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/replication/model"

	"github.com/stretchr/testify/require"
)

func Test_registryProxies(t *testing.T) {
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
							URL: "https://" + bareRegistry,
						},
					},
				},
			},
			expectedProxies: map[string]string{
				bareRegistry: "harbor.example.com/proxy-cache",
			},
		},
	}
	for _, testcase := range testcases {
		proxyMap := registryProxies("harbor.example.com", testcase.projects)
		require.Equal(t, testcase.expectedProxies, proxyMap, testcase.description)
	}
}
