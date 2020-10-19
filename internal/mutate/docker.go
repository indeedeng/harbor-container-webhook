package mutate

import (
	"fmt"
	"strings"

	"github.com/agext/regexp"
)

var dockerRegistry = regexp.MustCompile(`^(?P<registry>([a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*[\._]?:?[0-9]*)\/?(?P<imgname>.*)$`)

const BareRegistry = "registry.hub.docker.com"

// RegistryFromImageRef returns the registry (and port, if set) from the image reference,
// otherwise returns the default bare registry, "registry.hub.docker.com".
func RegistryFromImageRef(imageReference string) (registry string, err error) {
	if len(imageReference) > 0 {
		if !strings.Contains(imageReference, "/") {
			return BareRegistry, nil
		}
		matches := dockerRegistry.FindStringNamed(imageReference)
		// check if the reference has any private registry prefix
		if registry, ok := matches["registry"]; ok {
			return registry, nil
		}
	}
	// only possible if we were given nonsense
	return "", fmt.Errorf("image reference `%s` invalid, unable to parse registry or image name", imageReference)
}

func IsLibraryImage(imageReference string) bool {
	return !strings.Contains(imageReference, "/")
}

// ReplaceRegistryInImageRef returns the the image reference with the registry replaced.
func ReplaceRegistryInImageRef(imageReference, replacementRegistry string) (imageRef string, err error) {
	registry, err := RegistryFromImageRef(imageReference)
	if err != nil {
		return "", err
	}
	// special case for docker hub & bare image references
	if registry == BareRegistry && !strings.Contains(imageReference, BareRegistry) {
		// special case to add the "library" project to dockerhub std library image pulls
		if IsLibraryImage(imageReference) {
			return replacementRegistry + "/library/" + imageReference, nil
		}
		return replacementRegistry + "/" + imageReference, nil
	}
	matches := dockerRegistry.FindStringNamed(imageReference)
	// check if the reference has any private registry prefix
	if _, ok := matches["registry"]; ok {
		return fmt.Sprintf("%s/%s", replacementRegistry, matches["imgname"]), nil
	}
	// only possible if we were given nonsense
	return "", fmt.Errorf("image reference `%s` invalid, unable to replace registry", imageReference)
}
