harbor-container-webhook
=========

A kubernetes mutating webhook which rewrites container images to use a Harbor proxy cache.
This is typically useful for mirroring public registries that have low rate limits, such as dockerhub, or limiting
public bandwidth usage, by mirroring images in a local Harbor registry.

harbor-container-webhook inspects pod requests in a kubernetes cluster and rewrites the container image registry of
matching images.

* [Prerequisites](#prerequisites)
* [Installing](#installing)
* [Usage](#usage)
* [Local Development](#local-development)

Prerequisites
===
Requires kubernetes 1.17+ and can either be installed via helm or bare manifests.

Installing with helm
===

Option 1: Install from chart repository
```shell
helm repo add harbor-container-webhook https://indeedeng.github.io/harbor-container-webhook/

helm install harbor-container-webhook harbor-container-webhook/harbor-container-webhook -n harbor-container-webhook --create-namespace 
```

Option 2: Install chart from local build

Build and install the Helm chart locally after cloning the repository.
```shell
make helm.build

helm install harbor-container-webhook ./bin/chart/harbor-container-webhook.tgz -n harbor-container-webhook --create-namespace
```

Usage
===
The harbor-container-webhook rewrites are managed by configuration rules. Each rule contains a list of regular
expressions to match on, as well as an optional list of regular expressions to exclude. For each container image
reference which matches at least one match rule and none of the exclusion rules, then the registry is replaced
by the `replace` contents of the rule. If `checkUpstream` is enabled, the webhook will first fetch the manifest
the rewritten container image reference and verify it exists before rewriting the image.

Example configuration:
```yaml
port: 9443
certDir: "./hack/certs"
healthAddr: ":8080"
metricsAddr: ":8081"
rules:
  - name: 'docker.io rewrite rule'
    # image refs must match at least one of the rules, and not match any excludes
    matches:
      - '^docker.io'
    excludes:
      # for example, exclude ubuntu from harbor's proxy cache
      - '^docker.io/(library/)?ubuntu:.*$'
    replace: 'harbor.example.com/dockerhub-proxy'
    checkUpstream: false
  - name: 'docker.io ubuntu rewrite rule'
    # image refs must match at least one of the rules, and not match any excludes
    matches:
      - '^docker.io/(library/)?ubuntu:.*$'
    replace: 'harbor.example.com/ubuntu-proxy'
    checkUpstream: true # tests if the manifest for the rewritten image exists
    authSecretName: harbor-example-image-pull-secret # optional, defaults to "" - secret in the webhook namespace for authenticating to harbor.example.com
```
Local Development
===
`make help` prints out the help info for local development:

```
build         Build binary for the specified arch
docker.build  Build the docker image
fmt           ensure consistent code style
generate      Generate code
hack-test     curl the admission and no-op json bodies to the webhook
hack          build and run the webhook w/hack config
helm.build    Build helm chart
helm.docs     Generate helm docs
help          displays this help message
lint          run golangci-lint
test          Run tests
```

Ensure tests and linters pass with `make lint test`.

The webhook can be run locally with `make hack` and then `make hack-test` to submit sample responses to the webhook.

Contributing
===
We welcome contributions! Feel free to help make the harbor-container-webhook better.

Code of Conduct
===
harbor-container-webhook is governed by the [Contributer Covenant v1.4.1](CODE_OF_CONDUCT.md)

For more information please contact opensource@indeed.com.

License
===
The harbor-container-webhook is open source under the [Apache 2](LICENSE) license.
