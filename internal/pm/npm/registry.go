package npm

import "time"

type RegistryData struct {
	Name     string                      `json:"name"`
	DistTags map[string]string           `json:"dist-tags"`
	Versions map[string]*RegistryVersion `json:"versions"`
	// Keys include "created" and "modified"
	Time map[string]time.Time `json:"time"`
}

type RegistryVersion struct {
	*PackageJSON
	Deprecated string `json:"deprecated"`
	Dist       struct {
		Tarball      string `json:"tarball"`
		SHASum       string `json:"shasum"`
		Integrity    string `json:"integrity"`
		FileCount    int    `json:"fileCount"`
		UnpackedSize int64  `json:"unpackedSize"`
	} `json:"dist"`
}
