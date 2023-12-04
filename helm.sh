#!/bin/bash

set -euo pipefail

rm -rf deploy/
git checkout main -- deploy/
helm package deploy/charts/harbor-container-webhook
helm repo index . --url https://indeedeng.github.io/harbor-container-webhook --merge index.yaml
