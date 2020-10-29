package webhook

import (
	"strings"

	"github.com/containers/image/v5/docker/reference"
)

const BareRegistry = "docker.io"

// RegistryFromImageRef returns the registry (and port, if set) from the image reference,
// otherwise returns the default bare registry, "registry.hub.docker.com".
func RegistryFromImageRef(imageReference string) (registry string, err error) {
	ref, err := reference.ParseDockerRef(imageReference)
	if err != nil {
		return "", err
	}
	return reference.Domain(ref), nil
}

// ReplaceRegistryInImageRef returns the the image reference with the registry replaced.
func ReplaceRegistryInImageRef(imageReference, replacementRegistry string) (imageRef string, err error) {
	named, err := reference.ParseDockerRef(imageReference)
	if err != nil {
		return "", err
	}
	return strings.Replace(named.String(), reference.Domain(named), replacementRegistry, 1), nil
}
