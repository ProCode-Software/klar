package npm

type PackageJSON struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Keywords    []string          `json:"keywords"`
	License     string            `json:"license"`
	Author      any               `json:"author"` // string | { name: string }
	Scripts     map[string]string `json:"scripts"`
	Homepage    string            `json:"homepage"`
	Bugs        any               `json:"bugs"` // string | { url: string }
	Repository  struct {
		Type      string `json:"type"`
		URL       string `json:"url"`
		Directory string `json:"directory"`
	} `json:"repository"`
	Type            string            `json:"type"`
	Main            string            `json:"main"`
	Module          string            `json:"module"`
	DevDependencies map[string]string `json:"devDependencies"`
	Dependencies    map[string]string `json:"dependencies"`
	Exports         map[string]any    `json:"exports"` // string | [PackageExport]
}

type PackageExport struct{}
