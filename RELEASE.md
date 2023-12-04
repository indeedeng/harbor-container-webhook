# Release Process

TODO: Migrate these to github actions.

### Prerelease

1. Ensure the [CHANGELOG.md](CHANGELOG.md) is up to date.
2. Bump the [Chart.yaml](deploy/charts/harbor-container-webhook/Chart.yaml) `version` or `appVersion` as needed.

### Test the build of harbor-container-webhook

1. Build the webhook: `make docker.build`

### Release harbor-container-webhook

1. Release the webhook: `git tag 0.x.x && git push --tags`

### Release Helm Chart

1. Regenerate the helm chart + docs: `helm.build`