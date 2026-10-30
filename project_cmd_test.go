package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestProjectCommandsManageConfiguredFile(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "docs")
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	previousConfigFile := configFile
	configFile = filepath.Join(dir, "config", "glow.yml")
	t.Cleanup(func() { configFile = previousConfigFile })

	var output bytes.Buffer
	projectAddCmd.SetOut(&output)
	t.Cleanup(func() {
		projectAddCmd.SetOut(nil)
		projectListCmd.SetOut(nil)
		projectRemoveCmd.SetOut(nil)
	})
	if err := projectAddCmd.RunE(projectAddCmd, []string{"Docs", projectPath}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `Added project "Docs"`) {
		t.Fatalf("unexpected add output %q", output.String())
	}

	output.Reset()
	projectListCmd.SetOut(&output)
	if err := projectListCmd.RunE(projectListCmd, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != "Docs\t"+projectPath {
		t.Fatalf("unexpected list output %q", got)
	}

	names, directive := completeProjectNames(nil, nil, "")
	if len(names) != 1 || names[0] != "Docs" {
		t.Fatalf("unexpected completions %v", names)
	}
	if directive == 0 {
		t.Fatal("expected completion directive to disable file completion")
	}

	output.Reset()
	projectRemoveCmd.SetOut(&output)
	if err := projectRemoveCmd.RunE(projectRemoveCmd, []string{"Docs"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `Removed project "Docs"`) {
		t.Fatalf("unexpected remove output %q", output.String())
	}

	output.Reset()
	if err := projectListCmd.RunE(projectListCmd, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != "No projects configured." {
		t.Fatalf("unexpected empty list output %q", got)
	}
}

func TestProjectOpenRejectsMissingDirectory(t *testing.T) {
	dir := t.TempDir()
	previousConfigFile := configFile
	configFile = filepath.Join(dir, "glow.yml")
	t.Cleanup(func() { configFile = previousConfigFile })
	contents := `projects:
  - name: Missing
    path: /path/that/does/not/exist
`
	if err := os.WriteFile(configFile, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	err := projectOpenCmd.RunE(projectOpenCmd, []string{"Missing"})
	if err == nil || !strings.Contains(err.Error(), "unable to access project directory") {
		t.Fatalf("expected missing-directory error, got %v", err)
	}
}

func TestProjectCommandAddsFirstProjectInteractively(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "docs")
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	previousConfigFile := configFile
	configFile = filepath.Join(dir, "glow.yml")
	previousRunner := runProjectForCommand
	openedPath := ""
	runProjectForCommand = func(path, _ string) error {
		openedPath = path
		return nil
	}
	t.Cleanup(func() {
		configFile = previousConfigFile
		runProjectForCommand = previousRunner
		projectCmd.SetIn(nil)
		projectCmd.SetOut(nil)
	})

	var output bytes.Buffer
	projectCmd.SetIn(strings.NewReader("Docs\n" + projectPath + "\n"))
	projectCmd.SetOut(&output)
	if err := projectCmd.RunE(projectCmd, nil); err != nil {
		t.Fatal(err)
	}
	if openedPath != projectPath {
		t.Fatalf("expected project %q to open, got %q", projectPath, openedPath)
	}
	if !strings.Contains(output.String(), "Add your first project") ||
		!strings.Contains(output.String(), `Added project "Docs"`) {
		t.Fatalf("unexpected interactive output %q", output.String())
	}

	store := projectStore{path: configFile}
	projects, err := store.list()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Name != "Docs" || projects[0].Path != projectPath {
		t.Fatalf("unexpected stored projects %#v", projects)
	}
}

func TestProjectAddAcceptsInteractiveOrExplicitArguments(t *testing.T) {
	if err := projectAddCmd.Args(projectAddCmd, nil); err != nil {
		t.Fatalf("expected interactive mode to accept no arguments: %v", err)
	}
	if err := projectAddCmd.Args(projectAddCmd, []string{"Docs", "/docs"}); err != nil {
		t.Fatalf("expected explicit mode to accept two arguments: %v", err)
	}
	if err := projectAddCmd.Args(projectAddCmd, []string{"Docs"}); err == nil {
		t.Fatal("expected a single argument to be rejected")
	}
	for _, command := range []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "open", cmd: projectOpenCmd},
		{name: "remove", cmd: projectRemoveCmd},
	} {
		if err := command.cmd.Args(command.cmd, nil); err != nil {
			t.Fatalf("expected %s to accept no arguments: %v", command.name, err)
		}
		if err := command.cmd.Args(command.cmd, []string{"Docs"}); err != nil {
			t.Fatalf("expected %s to accept one argument: %v", command.name, err)
		}
	}
}

func TestProjectOpenWithoutNameUsesSelector(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "docs")
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	previousConfigFile := configFile
	configFile = filepath.Join(dir, "glow.yml")
	previousSelector := selectProjectForCommand
	previousRunner := runProjectForCommand
	selectProjectForCommand = func(projects []markdownProject) (*markdownProject, error) {
		return &projects[0], nil
	}
	openedPath := ""
	runProjectForCommand = func(path, _ string) error {
		openedPath = path
		return nil
	}
	t.Cleanup(func() {
		configFile = previousConfigFile
		selectProjectForCommand = previousSelector
		runProjectForCommand = previousRunner
	})

	store, err := configuredProjectStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.add("Docs", projectPath); err != nil {
		t.Fatal(err)
	}
	if err := projectOpenCmd.RunE(projectOpenCmd, nil); err != nil {
		t.Fatal(err)
	}
	if openedPath != projectPath {
		t.Fatalf("expected selected project %q to open, got %q", projectPath, openedPath)
	}
}

func TestProjectRemoveWithoutNameConfirmsSelection(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "docs")
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	previousConfigFile := configFile
	configFile = filepath.Join(dir, "glow.yml")
	previousSelector := selectProjectForCommand
	selectProjectForCommand = func(projects []markdownProject) (*markdownProject, error) {
		return &projects[0], nil
	}
	t.Cleanup(func() {
		configFile = previousConfigFile
		selectProjectForCommand = previousSelector
		projectRemoveCmd.SetIn(nil)
		projectRemoveCmd.SetOut(nil)
	})

	store, err := configuredProjectStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.add("Docs", projectPath); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	projectRemoveCmd.SetIn(strings.NewReader("yes\n"))
	projectRemoveCmd.SetOut(&output)
	if err := projectRemoveCmd.RunE(projectRemoveCmd, nil); err != nil {
		t.Fatal(err)
	}
	projects, err := store.list()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected selected project to be removed, got %#v", projects)
	}
	if !strings.Contains(output.String(), `Removed project "Docs"`) {
		t.Fatalf("unexpected remove output %q", output.String())
	}
}

func TestProjectRemoveWithoutNameCanBeCancelled(t *testing.T) {
	confirmed, err := confirmProjectRemoval(strings.NewReader("no\n"), &bytes.Buffer{}, "Docs")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed {
		t.Fatal("expected removal confirmation to reject no")
	}
}
