package solc

type Input struct {
	Language string              `json:"language,omitempty"`
	Sources  map[string]SourceIn `json:"sources,omitempty"`
	Settings Settings            `json:"settings,omitempty"`
}

type SourceIn struct {
	Keccak256 string `json:"keccak256,omitempty"`
	Content   string `json:"content,omitempty"`
}

type Settings struct {
	Remappings      []string                       `json:"remappings,omitempty"`
	Optimizer       Optimizer                      `json:"optimizer,omitempty"`
	EVMVersion      string                         `json:"evmVersion,omitempty"`
	OutputSelection map[string]map[string][]string `json:"outputSelection,omitempty"`
}

type Optimizer struct {
	Enabled bool `json:"enabled,omitempty"`
	Runs    int  `json:"runs,omitempty"`
}
