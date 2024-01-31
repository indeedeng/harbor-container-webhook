package webhook

// slimManifest is a partial representation of the oci manifest to access the mediaType
type slimManifest struct {
	MediaType string `json:"mediaType"`
}

type platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

// indexManifest is a partial representation of the sub manifest present in a manifest list
type indexManifest struct {
	MediaType string   `json:"mediaType"`
	Platform  platform `json:"platform"`
}

// slimManifestList is a partial representation of the oci manifest list to access the supported architectures
type slimManifestList struct {
	MediaType string          `json:"mediaType"`
	Manifests []indexManifest `json:"manifests"`
}
