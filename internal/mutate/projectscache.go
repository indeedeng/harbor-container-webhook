package mutate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/goharbor/harbor/src/common/models"

	"github.com/peterhellberg/link"

	ctrl "sigs.k8s.io/controller-runtime"
)

var logger = ctrl.Log.WithName("projects-cache")

const defaultPageSize = 20

type projectWithSummary struct {
	models.Project
	models.ProjectSummary //nolint:govet
}

type projectsCache struct {
	client         *http.Client
	harborEndpoint string
	authHeader     string
	pageSize       int
	resyncInterval time.Duration

	lock struct {
		sync.RWMutex
		projects   []projectWithSummary
		expiration time.Time
		updating   bool
	}
}

func NewProjectsCache(client *http.Client, harborEndpoint, harborUser, harborPass string, resyncInterval time.Duration) ProjectsCache {
	return &projectsCache{
		client:         client,
		harborEndpoint: harborEndpoint,
		authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte(harborUser+":"+harborPass)),
		resyncInterval: resyncInterval,
	}
}

var _ ProjectsCache = (*projectsCache)(nil)

type ProjectsCache interface {
	List() ([]projectWithSummary, error)
}

func (p *projectsCache) List() ([]projectWithSummary, error) {
	projects, err := p.listAll()
	if err != nil {
		return []projectWithSummary{}, err
	}
	// TODO: cache
	return p.enrichProjects(projects)
}

func (p *projectsCache) updateInProgress() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.lock.updating
}

func (p *projectsCache) updateCache() {
	p.lock.Lock()
	p.lock.updating = true
	p.lock.Unlock()
	defer func() {
		p.lock.Lock()
		p.lock.updating = false
		p.lock.Unlock()
	}()
	projects, err := p.listAll()
	if err != nil {
		logger.Error(err, "failed to list all harbor projects")
	}
	// TODO: cache
	return p.enrichProjects(projects)
}

func (p *projectsCache) enrichProjects(projects []models.Project) ([]projectWithSummary, error) {
	summaries := make([]projectWithSummary, 0, len(projects))
	for i, project := range projects {
		url := fmt.Sprintf("%s/api/v2.0/projects/%d/summary", p.harborEndpoint, project.ProjectID)
		bytes, _, err := p.httpGet(url)
		if err != nil {
			return []projectWithSummary{}, err
		}
		var projectSummary models.ProjectSummary
		if err := json.Unmarshal(bytes, &projectSummary); err != nil {
			logger.Error(err, "failed to unmarshal project summary for "+project.Name)
			return []projectWithSummary{}, err
		}
		summaries = append(summaries, projectWithSummary{
			Project:        projects[i],
			ProjectSummary: projectSummary,
		})
	}
	return summaries, nil
}

func (p *projectsCache) listAll() ([]models.Project, error) {
	list := make([]models.Project, 0)
	projects, group, err := p.fetchFirstProjects()
	if err != nil {
		return []models.Project{}, err
	}
	list = append(list, projects...)
	for k, l := range group {
		for l.Rel == "next" {
			projects, group, err = p.fetchProjects(l.URI)
			list = append(list, projects...)
			l = group[k]
		}
	}
	return list, nil
}

func (p *projectsCache) fetchFirstProjects() ([]models.Project, link.Group, error) {
	url := fmt.Sprintf("%s/api/v2.0/projects?page=%d&page_size=%d", p.harborEndpoint, 1, p.pageSize)
	return p.fetchProjects(url)
}

func (p *projectsCache) fetchProjects(url string) ([]models.Project, link.Group, error) {
	bytes, headers, err := p.httpGet(url)
	if err != nil {
		return []models.Project{}, link.Group{}, err
	}
	var projects []models.Project
	if err := json.Unmarshal(bytes, &projects); err != nil {
		logger.Error(err, "failed to unmarshal response from "+url)
		return []models.Project{}, link.Group{}, err
	}
	return projects, link.ParseHeader(headers), nil
}

func (p *projectsCache) httpGet(url string) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, url, nil)
	if err != nil {
		logger.Error(err, "failed to create http.Request for "+url)
		return []byte{}, nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", p.authHeader)

	response, err := p.client.Do(req)
	if err != nil {
		logger.Error(err, "failed to get "+url)
		return []byte{}, nil, err
	}
	if response.Body == nil {
		return []byte{}, nil, errors.New("no response body in request to " + url)
	}
	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Error(err, "failed to read response body from "+url)
		return []byte{}, nil, err
	}
	return bytes, response.Header, nil
}
