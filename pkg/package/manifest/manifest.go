package manifest

import "encoding/json"

type Platform string

const (
	PlatformJS   Platform = "js"
	PlatformKlar Platform = "klar"
)

type Package struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Version     string     `json:"version"`
	Author      string     `json:"author"`
	License     string     `json:"license"`
	Platform    []Platform `json:"platform"`
	KlarVersion string     `json:"klar"`

	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`

	Packages []Package `json:"packages"`
}

func ParseManifest(manifest string) (*Package, error) {
	pkg := &Package{}
	err := json.Unmarshal([]byte(manifest), pkg)
	if err != nil {
		return nil, err
	}
	return pkg, nil
}
