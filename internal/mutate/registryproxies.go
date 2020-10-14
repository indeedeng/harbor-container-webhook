package mutate

import (
	"net/url"
)

func registryProxies(harborEndpoint string, projects []projectWithSummary) map[string]string {
	proxyMap := make(map[string]string)
	for _, project := range projects {
		if project.Registry != nil && project.Registry.URL != "" {
			url, err := url.Parse(project.Registry.URL)
			if err != nil {
				logger.Error(err, "failed to parse url: "+project.Registry.URL)
				continue
			}
			if _, ok := proxyMap[url.Host]; !ok {
				proxyMap[url.Host] = harborEndpoint + "/" + project.Name
			}
		}
	}
	return proxyMap
}
