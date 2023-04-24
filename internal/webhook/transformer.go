package webhook

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/indeedeng-alpha/harbor-container-webhook/internal/config"

	"github.com/prometheus/client_golang/prometheus"

	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	rewrite = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hcw",
		Subsystem: "rules",
		Name:      "rewrite_success",
		Help:      "image rewrite success metrics for this rule",
	}, []string{"name"})
	rewriteTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "hcw",
		Subsystem: "rules",
		Name:      "rewrite_duration_seconds",
		Help:      "image rewrite duration distribution for this rule",
	}, []string{"name"})
	rewriteErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hcw",
		Subsystem: "rules",
		Name:      "rewrite_errors",
		Help:      "errors while parsing and rewriting images for this rule",
	}, []string{"name"})
	upstream = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hcw",
		Subsystem: "rules",
		Name:      "upstream_checks",
		Help:      "image rewrite upstream checks that succeeded this rule",
	}, []string{"name"})
	upstreamErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hcw",
		Subsystem: "rules",
		Name:      "upstream_check_errors",
		Help:      "image rewrite upstream checks that errored for this rule",
	}, []string{"name"})
)

func init() {
	metrics.Registry.MustRegister(rewrite, rewriteTime, rewriteErrors, upstream, upstreamErrors)
}

var invalidMetricChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)

type multiTransformer struct {
	transformers []*ruleTransformer
}

func NewMultiTransformer(rules []config.ProxyRule) (ContainerTransformer, error) {
	transformers := make([]*ruleTransformer, 0, len(rules))
	for _, rule := range rules {
		transformer, err := newRuleTransformer(rule)
		if err != nil {
			return nil, err
		}
		transformers = append(transformers, transformer)
	}
	return &multiTransformer{transformers: transformers}, nil
}

func (t *multiTransformer) RewriteImage(imageRef, platform, os string) (string, error) {
	for _, transformer := range t.transformers {
		updatedRef, err := transformer.RewriteImage(imageRef, platform, os)
		if err != nil {
			return "", fmt.Errorf("transformer %q failed to update imageRef %q: %w", transformer.rule.Name, imageRef, err)
		}
		if updatedRef != imageRef {
			if transformer.rule.CheckUpstream {
				if err := transformer.checkUpstream(updatedRef, &v1.Platform{Architecture: platform, OS: os}); err != nil {
					logger.Info(fmt.Sprintf("skipping rewriting %q to %q, could not fetch image manifest: %s", imageRef, updatedRef, err.Error()))
					continue
				}
			}
			return updatedRef, nil
		}
	}
	return imageRef, nil
}

type ruleTransformer struct {
	rule       config.ProxyRule
	metricName string

	matches  []*regexp.Regexp
	excludes []*regexp.Regexp
}

var _ ContainerTransformer = (*ruleTransformer)(nil)

func newRuleTransformer(rule config.ProxyRule) (*ruleTransformer, error) {
	transformer := &ruleTransformer{
		rule:       rule,
		metricName: invalidMetricChars.ReplaceAllString(strings.ToLower(rule.Name), "_"),
		matches:    make([]*regexp.Regexp, 0, len(rule.Matches)),
		excludes:   make([]*regexp.Regexp, 0, len(rule.Excludes)),
	}
	for _, matchRegex := range rule.Matches {
		matcher, err := regexp.Compile(matchRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to compile regex %q: %w", matchRegex, err)
		}
		transformer.matches = append(transformer.matches, matcher)
	}
	for _, excludeRegex := range rule.Excludes {
		excluder, err := regexp.Compile(excludeRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to compile exclude regex %q: %w", excludeRegex, err)
		}
		transformer.excludes = append(transformer.excludes, excluder)
	}

	return transformer, nil
}

func (t *ruleTransformer) checkUpstream(imageRef string, platform *v1.Platform) error {
	if _, err := crane.Manifest(imageRef, crane.WithPlatform(platform)); err != nil {
		upstreamErrors.WithLabelValues(t.metricName).Inc()
		return err
	}
	upstream.WithLabelValues(t.metricName).Inc()
	return nil
}

func (t *ruleTransformer) RewriteImage(imageRef, _, _ string) (string, error) {
	start := time.Now()
	updatedRef, err := t.rewriteImage(imageRef)
	duration := time.Since(start)
	if err != nil {
		rewriteErrors.WithLabelValues(t.metricName).Inc()
	} else {
		rewrite.WithLabelValues(t.metricName).Inc()
		rewriteTime.WithLabelValues(t.metricName).Observe(duration.Seconds())
	}
	return updatedRef, err
}

func (t *ruleTransformer) rewriteImage(imageRef string) (string, error) {
	registry, err := RegistryFromImageRef(imageRef)
	if err != nil {
		rewriteErrors.WithLabelValues(t.metricName).Inc()
		return "", err
	}
	// shenanigans to get a fully normalized ref, e.g 'ubuntu' -> 'docker.io/library/ubuntu:latest'
	normalizedRef, err := ReplaceRegistryInImageRef(imageRef, registry)
	if err != nil {
		rewriteErrors.WithLabelValues(t.metricName).Inc()
		return "", err
	}

	if t.findMatch(normalizedRef) && !t.anyExclusion(normalizedRef) {
		return ReplaceRegistryInImageRef(imageRef, t.rule.Replace)
	}

	return imageRef, nil
}

func (t *ruleTransformer) findMatch(imageRef string) bool {
	for _, rule := range t.matches {
		if rule.MatchString(imageRef) {
			return true
		}
	}
	return false
}

func (t *ruleTransformer) anyExclusion(imageRef string) bool {
	for _, rule := range t.excludes {
		if rule.MatchString(imageRef) {
			return true
		}
	}
	return false
}
