package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectStoreCRUDPreservesConfig(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first")
	secondPath := filepath.Join(dir, "second")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	configPath := filepath.Join(dir, "glow.yml")
	const original = `# keep this comment
style: "auto"
width: 91
projects: []
`
	if err := os.WriteFile(configPath, []byte(original), 0o640); err != nil {
		t.Fatal(err)
	}
	store := projectStore{path: configPath}

	first, err := store.add("First Project", firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != firstPath {
		t.Fatalf("expected absolute path %q, got %q", firstPath, first.Path)
	}
	if _, err := store.add("Second Project", secondPath); err != nil {
		t.Fatal(err)
	}
	if _, err := store.add("First Project", firstPath); err == nil {
		t.Fatal("expected duplicate project name to fail")
	}

	projects, err := store.list()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 || projects[0].Name != "First Project" || projects[1].Name != "Second Project" {
		t.Fatalf("expected projects to preserve insertion order, got %#v", projects)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	if !strings.Contains(contents, "# keep this comment") || !strings.Contains(contents, "width: 91") {
		t.Fatalf("expected unrelated configuration and comments to be preserved, got:\n%s", contents)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("expected config permissions 0640, got %04o", info.Mode().Perm())
	}

	removed, err := store.remove("First Project")
	if err != nil {
		t.Fatal(err)
	}
	if removed.Name != "First Project" {
		t.Fatalf("expected to remove First Project, got %q", removed.Name)
	}
	if _, err := store.find("First Project"); err == nil {
		t.Fatal("expected removed project lookup to fail")
	}
	if _, err := store.remove("Missing"); err == nil {
		t.Fatal("expected missing project removal to fail")
	}

	temporaryFiles, err := filepath.Glob(filepath.Join(dir, ".glow-projects-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("expected temporary files to be cleaned up, got %v", temporaryFiles)
	}
}

func TestProjectStoreRejectsInvalidProjects(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "glow.yml")
	if err := os.WriteFile(configPath, []byte("projects: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := projectStore{path: configPath}

	if _, err := store.add("", dir); err == nil {
		t.Fatal("expected empty project name to fail")
	}
	if _, err := store.add("Missing", filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected missing project directory to fail")
	}
	filePath := filepath.Join(dir, "file.md")
	if err := os.WriteFile(filePath, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.add("File", filePath); err == nil {
		t.Fatal("expected file project path to fail")
	}
}

func TestProjectStoreAddsProjectsToExistingConfig(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "docs")
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "glow.yml")
	if err := os.WriteFile(configPath, []byte("# existing config\nwidth: 120\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := projectStore{path: configPath}
	if _, err := store.add("Docs", projectPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	for _, expected := range []string{
		"# existing config",
		"width: 120",
		`# Markdown project directories managed by "glow project".`,
		"projects:",
		"name: Docs",
	} {
		if !strings.Contains(contents, expected) {
			t.Fatalf("expected config to contain %q, got:\n%s", expected, contents)
		}
	}
}

func TestProjectStoreRejectsAmbiguousConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "glow.yml")
	contents := `projects:
  - name: duplicate
    path: /first
  - name: duplicate
    path: /second
`
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := (projectStore{path: configPath}).list(); err == nil {
		t.Fatal("expected duplicate configured names to fail")
	}
}
