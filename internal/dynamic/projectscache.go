package dynamic

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

	"golang.org/x/sync/semaphore"

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
	}
	updating *semaphore.Weighted
}

func NewProjectsCache(client *http.Client, harborEndpoint, harborUser, harborPass string, resyncInterval time.Duration) ProjectsCache {
	return &projectsCache{
		client:         client,
		harborEndpoint: harborEndpoint,
		authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte(harborUser+":"+harborPass)),
		resyncInterval: resyncInterval,
		updating:       semaphore.NewWeighted(1),
		pageSize:       defaultPageSize,
	}
}

var _ ProjectsCache = (*projectsCache)(nil)

type ProjectsCache interface {
	List() ([]projectWithSummary, error)
}

func (p *projectsCache) List() ([]projectWithSummary, error) {
	if !p.cacheValid() {
		logger.Info("cache out of date, serving stale projects")
		go func() {
			if err := p.updateCache(); err != nil {
				logger.Error(err, "failed to update projects cache")
			} else {
				logger.Info("projects cache updated")
			}
		}()
	}
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.lock.projects, nil
}

func (p *projectsCache) cacheValid() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return !p.lock.expiration.IsZero() && time.Now().Before(p.lock.expiration)
}

func (p *projectsCache) updateCache() error {
	if !p.updating.TryAcquire(1) {
		return nil
	}
	defer p.updating.Release(1)

	ctx := context.Background()
	projects, err := p.listAll(ctx)
	if err != nil {
		return err
	}
	projectsWithSummary, err := p.enrichProjects(ctx, projects)
	if err != nil {
		return err
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.lock.projects = projectsWithSummary
	p.lock.expiration = time.Now().Add(p.resyncInterval)
	return nil
}

func (p *projectsCache) enrichProjects(ctx context.Context, projects []models.Project) ([]projectWithSummary, error) {
	summaries := make([]projectWithSummary, 0, len(projects))
	for i, project := range projects {
		url := fmt.Sprintf("%s/api/v2.0/projects/%d/summary", p.harborEndpoint, project.ProjectID)
		bytes, _, err := p.httpGet(ctx, url)
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

func (p *projectsCache) listAll(ctx context.Context) ([]models.Project, error) {
	list := make([]models.Project, 0)
	projects, group, err := p.fetchFirstProjects(ctx)
	if err != nil {
		return []models.Project{}, fmt.Errorf("failed to list projects: %w", err)
	}
	list = append(list, projects...)
	for k, l := range group {
		for l != nil && l.Rel == "next" {
			projects, group, err = p.fetchProjects(ctx, l.URI)
			if err != nil {
				return []models.Project{}, fmt.Errorf("failed to list projects from %q: %w", l.URI, err)
			}
			list = append(list, projects...)
			l = group[k]
		}
	}
	return list, nil
}

func (p *projectsCache) fetchFirstProjects(ctx context.Context) ([]models.Project, link.Group, error) {
	url := fmt.Sprintf("%s/api/v2.0/projects?page=%d&page_size=%d", p.harborEndpoint, 1, p.pageSize)
	return p.fetchProjects(ctx, url)
}

func (p *projectsCache) fetchProjects(ctx context.Context, url string) ([]models.Project, link.Group, error) {
	bytes, headers, err := p.httpGet(ctx, url)
	if err != nil {
		return []models.Project{}, link.Group{}, err
	}
	var projects []models.Project
	if err := json.Unmarshal(bytes, &projects); err != nil {
		return []models.Project{}, link.Group{}, fmt.Errorf("failed to unmarshal response from %q: %w", url, err)
	}
	return projects, link.ParseHeader(headers), nil
}

func (p *projectsCache) httpGet(ctx context.Context, url string) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return []byte{}, nil, fmt.Errorf("failed to create http.Request for %q: %w", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", p.authHeader)

	response, err := p.client.Do(req)
	if err != nil {
		return []byte{}, nil, fmt.Errorf("failed to get %q: %w", url, err)
	}
	if response.Body == nil {
		return []byte{}, nil, errors.New("no response body in request to " + url)
	}
	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []byte{}, nil, fmt.Errorf("failed to read response body from %q: %w", url, err)
	}
	return bytes, response.Header, nil
}
