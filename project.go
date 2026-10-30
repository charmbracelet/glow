package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"charm.land/glow/v3/utils"
	"go.yaml.in/yaml/v3"
)

type markdownProject struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type projectConfig struct {
	Projects []markdownProject `yaml:"projects"`
}

type projectStore struct {
	path string
}

func configuredProjectStore() (projectStore, error) {
	if err := ensureConfigFile(); err != nil {
		return projectStore{}, err
	}
	return projectStore{path: configFile}, nil
}

func (s projectStore) list() ([]markdownProject, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("unable to read project configuration: %w", err)
	}

	var cfg projectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unable to parse project configuration: %w", err)
	}
	if err := validateProjects(cfg.Projects); err != nil {
		return nil, err
	}
	return cfg.Projects, nil
}

func (s projectStore) find(name string) (markdownProject, error) {
	projects, err := s.list()
	if err != nil {
		return markdownProject{}, err
	}
	for _, project := range projects {
		if project.Name == name {
			return project, nil
		}
	}
	return markdownProject{}, fmt.Errorf("project %q is not configured", name)
}

func (s projectStore) add(name, path string) (markdownProject, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return markdownProject{}, errors.New("project name cannot be empty")
	}

	path, err := normalizeProjectPath(path)
	if err != nil {
		return markdownProject{}, err
	}
	projects, err := s.list()
	if err != nil {
		return markdownProject{}, err
	}
	for _, project := range projects {
		if project.Name == name {
			return markdownProject{}, fmt.Errorf("project %q is already configured", name)
		}
	}

	project := markdownProject{Name: name, Path: path}
	projects = append(projects, project)
	if err := s.write(projects); err != nil {
		return markdownProject{}, err
	}
	return project, nil
}

func (s projectStore) remove(name string) (markdownProject, error) {
	projects, err := s.list()
	if err != nil {
		return markdownProject{}, err
	}
	for i, project := range projects {
		if project.Name == name {
			projects = append(projects[:i], projects[i+1:]...)
			if err := s.write(projects); err != nil {
				return markdownProject{}, err
			}
			return project, nil
		}
	}
	return markdownProject{}, fmt.Errorf("project %q is not configured", name)
}

func (s projectStore) write(projects []markdownProject) error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("unable to read project configuration: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("unable to parse project configuration: %w", err)
	}
	root, err := yamlMappingRoot(&document)
	if err != nil {
		return err
	}

	var value yaml.Node
	if err := value.Encode(projects); err != nil {
		return fmt.Errorf("unable to encode projects: %w", err)
	}
	if !replaceYAMLMappingValue(root, "projects", &value) {
		root.Content = append(root.Content,
			&yaml.Node{
				Kind:        yaml.ScalarNode,
				Tag:         "!!str",
				Value:       "projects",
				HeadComment: "Markdown project directories managed by \"glow project\".",
			},
			&value,
		)
	}

	mode := fs.FileMode(0o600)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := writeYAMLAtomically(s.path, &document, mode); err != nil {
		return fmt.Errorf("unable to write project configuration: %w", err)
	}
	return nil
}

func validateProjects(projects []markdownProject) error {
	seen := make(map[string]struct{}, len(projects))
	for i, project := range projects {
		if strings.TrimSpace(project.Name) == "" {
			return fmt.Errorf("project %d has an empty name", i+1)
		}
		if strings.TrimSpace(project.Path) == "" {
			return fmt.Errorf("project %q has an empty path", project.Name)
		}
		if _, ok := seen[project.Name]; ok {
			return fmt.Errorf("project %q is configured more than once", project.Name)
		}
		seen[project.Name] = struct{}{}
	}
	return nil
}

func normalizeProjectPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("project path cannot be empty")
	}
	path, err := filepath.Abs(utils.ExpandPath(path))
	if err != nil {
		return "", fmt.Errorf("unable to resolve project path: %w", err)
	}
	if err := validateProjectDirectory(path); err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func validateProjectDirectory(path string) error {
	info, err := os.Stat(utils.ExpandPath(path))
	if err != nil {
		return fmt.Errorf("unable to access project directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project path %q is not a directory", path)
	}
	return nil
}

func yamlMappingRoot(document *yaml.Node) (*yaml.Node, error) {
	if len(document.Content) == 0 {
		root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		document.Kind = yaml.DocumentNode
		document.Content = []*yaml.Node{root}
		return root, nil
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, errors.New("glow configuration must be a YAML mapping")
	}
	return root, nil
}

func replaceYAMLMappingValue(root *yaml.Node, key string, value *yaml.Node) bool {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = value
			return true
		}
	}
	return false
}

func writeYAMLAtomically(path string, document *yaml.Node, mode fs.FileMode) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".glow-projects-*")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		_ = encoder.Close()
		_ = file.Close()
		return err
	}
	if err := encoder.Close(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
