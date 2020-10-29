package webhook

import (
	"fmt"
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

func IsLibraryImage(imageReference string) bool {
	return !strings.Contains(imageReference, "/")
}

// ReplaceRegistryInImageRef returns the the image reference with the registry replaced.
func ReplaceRegistryInImageRef(imageReference, replacementRegistry string) (imageRef string, err error) {
	named, err := reference.ParseDockerRef(imageReference)
	if err != nil {
		return "", err
	}

	// special case for docker hub & bare image references
	// see: https://github.com/containers/image/blob/v5.7.0/docker/reference/normalize.go#L100
	if reference.Domain(named) == BareRegistry && !strings.ContainsRune(reference.Path(named), '/') {
		if canonical, ok := named.(reference.Canonical); ok {
			return fmt.Sprintf("%s/library/%s@%s", replacementRegistry, reference.Path(canonical), canonical.Digest().String()), nil
		}
		if taggedName, ok := named.(reference.NamedTagged); ok {
			return fmt.Sprintf("%s/library/%s:%s", replacementRegistry, reference.Path(taggedName), taggedName.Tag()), nil
		}
		return replacementRegistry + "/library/" + reference.Path(named), nil
	}

	if canonical, ok := named.(reference.Canonical); ok {
		return fmt.Sprintf("%s/%s@%s", replacementRegistry, reference.Path(canonical), canonical.Digest().String()), nil
	}
	if taggedName, ok := named.(reference.NamedTagged); ok {
		return fmt.Sprintf("%s/%s:%s", replacementRegistry, reference.Path(taggedName), taggedName.Tag()), nil
	}
	return replacementRegistry + "/" + reference.Path(named), nil
}
