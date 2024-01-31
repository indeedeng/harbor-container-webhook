package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/containerd/containerd/images"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"

	"github.com/indeedeng-alpha/harbor-container-webhook/internal/config"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/prometheus/client_golang/prometheus"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
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

// ContainerTransformer rewrites docker image references for harbor proxy cache projects.
type ContainerTransformer interface {
	// Name returns the name of the transformer rule
	Name() string

	// RewriteImage takes a docker image reference and returns the same image reference rewritten for a harbor
	// proxy cache project endpoint, if one is available, else returns the original image reference.
	RewriteImage(imageRef string) (string, error)

	// CheckUpstream ensures that the docker image reference exists in the upstream registry
	// and returns if the image exists, or an error if the registry can't be contacted.
	CheckUpstream(ctx context.Context, imageRef string) (bool, error)
}

func MakeTransformers(rules []config.ProxyRule, client client.Client) ([]ContainerTransformer, error) {
	transformers := make([]ContainerTransformer, 0, len(rules))
	for _, rule := range rules {
		transformer, err := newRuleTransformer(rule)
		transformer.client = client
		if err != nil {
			return nil, err
		}
		transformers = append(transformers, transformer)
	}
	return transformers, nil
}

type ruleTransformer struct {
	rule       config.ProxyRule
	metricName string

	client client.Client

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

func (t *ruleTransformer) Name() string {
	return t.rule.Name
}

func (t *ruleTransformer) CheckUpstream(ctx context.Context, imageRef string) (bool, error) {
	if !t.rule.CheckUpstream {
		return true, nil
	}

	options := make([]crane.Option, 0)
	if t.rule.AuthSecretName != "" {
		auth, err := t.auth(ctx)
		if err != nil {
			return false, err
		}
		options = append(options, crane.WithAuth(auth))
	}
	// we don't pass in the platform to crane to retrieve the full manifest list for multi-arch
	options = append(options, crane.WithContext(ctx))
	manifestBytes, err := crane.Manifest(imageRef, options...)
	if err != nil {
		upstreamErrors.WithLabelValues(t.metricName).Inc()
		return false, err
	}

	// try and parse the manifest to decode the MediaType to determine if it's a manifest or manifest list
	manifest := slimManifest{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		upstreamErrors.WithLabelValues(t.metricName).Inc()
		return false, fmt.Errorf("failed to parse manifest %s payload=%s: %w", imageRef, string(manifestBytes), err)
	}

	switch manifest.MediaType {
	case images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:
		manifestList := slimManifestList{}
		if err := json.Unmarshal(manifestBytes, &manifestList); err != nil {
			upstreamErrors.WithLabelValues(t.metricName).Inc()
			return false, fmt.Errorf("failed to parse manifest list %s, payload=%s: %w", imageRef, string(manifestBytes), err)
		}
		matches := 0
		for _, rulePlatform := range t.rule.Platforms {
			for _, subManifest := range manifestList.Manifests {
				subPlatform := subManifest.Platform.OS + "/" + subManifest.Platform.Architecture
				if subPlatform == rulePlatform {
					matches++
					break
				}
			}
		}
		if matches == len(t.rule.Platforms) {
			upstream.WithLabelValues(t.metricName).Inc()
			return true, nil
		}

		return false, nil
	case images.MediaTypeDockerSchema1Manifest, images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest:
		upstream.WithLabelValues(t.metricName).Inc()
		return true, nil
	default:
		logger.Info(fmt.Sprintf("unknown manifest media type: %s, rule=%s,imageRef=%s", manifest.MediaType, t.rule.Name, imageRef))
		upstream.WithLabelValues(t.metricName).Inc()
		return true, nil
	}
}

func (t *ruleTransformer) auth(ctx context.Context) (authn.Authenticator, error) {
	var secret corev1.Secret
	logger.Info("token key: ", "key", client.ObjectKey{Namespace: t.rule.Namespace, Name: t.rule.AuthSecretName})
	if err := t.client.Get(ctx, client.ObjectKey{Namespace: t.rule.Namespace, Name: t.rule.AuthSecretName}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %q for upstream manifests: %w", t.rule.AuthSecretName, err)
	}

	if dockerConfigJSONBytes, dockerConfigJSONExists := secret.Data[corev1.DockerConfigJsonKey]; (secret.Type == corev1.SecretTypeDockerConfigJson) && dockerConfigJSONExists && (len(dockerConfigJSONBytes) > 0) {
		dockerConfigJSON := DockerConfigJSON{}
		if err := json.Unmarshal(dockerConfigJSONBytes, &dockerConfigJSON); err != nil {
			return nil, err
		}
		// TODO: full keyring support?
		if len(dockerConfigJSON.Auths) != 1 {
			return nil, fmt.Errorf("only .dockerconfigjson with one auth method is supported, found %d", len(dockerConfigJSON.Auths))
		}
		for _, method := range dockerConfigJSON.Auths {
			if method.Auth != "" {
				user, pass, err := decodeDockerConfigFieldAuth(method.Auth)
				if err != nil {
					return nil, fmt.Errorf("failed to parse auth docker config auth field in secret %q", t.rule.AuthSecretName)
				}
				return &authn.Basic{Username: user, Password: pass}, nil
			}
			return &authn.Basic{Username: method.Username, Password: method.Password}, nil
		}
	}

	return nil, fmt.Errorf("failed to parse auth secret %q, no docker config found", t.rule.AuthSecretName)
}

func (t *ruleTransformer) RewriteImage(imageRef string) (string, error) {
	start := time.Now()
	rewritten, updatedRef, err := t.doRewriteImage(imageRef)
	duration := time.Since(start)
	if err != nil {
		rewriteErrors.WithLabelValues(t.metricName).Inc()
	} else if rewritten {
		rewrite.WithLabelValues(t.metricName).Inc()
		rewriteTime.WithLabelValues(t.metricName).Observe(duration.Seconds())
	}
	return updatedRef, err
}

func (t *ruleTransformer) doRewriteImage(imageRef string) (rewritten bool, updatedRef string, err error) {
	registry, err := RegistryFromImageRef(imageRef)
	if err != nil {
		return false, "", err
	}
	// shenanigans to get a fully normalized ref, e.g 'ubuntu' -> 'docker.io/library/ubuntu:latest'
	normalizedRef, err := ReplaceRegistryInImageRef(imageRef, registry)
	if err != nil {
		return false, "", err
	}

	if t.findMatch(normalizedRef) && !t.anyExclusion(normalizedRef) {
		updatedRef, err = ReplaceRegistryInImageRef(imageRef, t.rule.Replace)
		return true, updatedRef, err
	}

	return false, imageRef, nil
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
