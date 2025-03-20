package git

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	MULTI_MERGE_MANIFEST_FILENAME = ".pila_multi_merge.yaml"

	MULTI_MERGE_MANIFEST_TYPE_BRANCHES = "branches"
	MULTI_MERGE_MANIFEST_TYPE_LABELS   = "labels"
)

type MultiMergeManifest struct {
	MainSha    string                `yaml:"main_sha"`
	Target     string                `yaml:"target"`
	Type       string                `yaml:"type"`
	References []MultiMergeReference `yaml:"references"`
}

type MultiMergeReference struct {
	Name   string `yaml:"name"`
	Merged bool   `yaml:"merged"`
	Note   string `yaml:"note,omitempty"`
}

func LoadMultiMergeManifest() (*MultiMergeManifest, error) {
	// Read YAML file
	data, err := os.ReadFile(MULTI_MERGE_MANIFEST_FILENAME)
	if err != nil {
		return nil, err
	}

	var manifest MultiMergeManifest
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

// Are all branches merged
func (m *MultiMergeManifest) IsDone() bool {
	for _, reference := range m.References {
		if !reference.Merged {
			return false
		}
	}

	return true
}

// Mark all branches as un-merged
func (m *MultiMergeManifest) Reset() bool {
	for i := range m.References {
		reference := &m.References[i]
		reference.Merged = false
	}

	m.Save()
	return true
}

// Save manifest to disk
func (m *MultiMergeManifest) Save() error {
	data, err := yaml.Marshal(m)
	if err != nil {
		panic(err)
	}
	os.WriteFile(MULTI_MERGE_MANIFEST_FILENAME, data, 0o644)

	return nil
}

func (m *MultiMergeManifest) Remove() error {
	return os.Remove(MULTI_MERGE_MANIFEST_FILENAME)
}
