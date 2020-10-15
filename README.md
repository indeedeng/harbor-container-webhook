# harbor-container-webhook
=========

# Project Overview

harbor-container-webhook is a kubernetes mutating webhook controller which inspects pod requests to a kubernetes
cluster and replaces container image registry with the same image from the harbor proxy-cache if a 
proxy-cache for the image reference registry exists.

# License

The harbor-container-webhooks is open source under the [Apache 2](LICENSE) license.
