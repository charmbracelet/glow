package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

var (
	selectProjectForCommand = selectProject
	runProjectForCommand    = runTUI
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage and open Markdown projects",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		projects, err := store.list()
		if err != nil {
			return err
		}
		if len(projects) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No projects configured. Add your first project:")
			project, err := addProjectInteractively(cmd, store)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added project %q: %s\n", project.Name, project.Path)
			return openProject(project)
		}

		selected, err := selectProjectForCommand(projects)
		if err != nil {
			return err
		}
		if selected == nil {
			return nil
		}
		return openProject(*selected)
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured Markdown projects",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		projects, err := store.list()
		if err != nil {
			return err
		}
		if len(projects) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No projects configured.")
			return nil
		}
		for _, project := range projects {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", project.Name, project.Path)
		}
		return nil
	},
}

var projectAddCmd = &cobra.Command{
	Use:   "add [NAME PATH]",
	Short: "Add a Markdown project",
	Args:  zeroOrExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		var project markdownProject
		if len(args) == 0 {
			project, err = addProjectInteractively(cmd, store)
		} else {
			project, err = store.add(args[0], args[1])
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added project %q: %s\n", project.Name, project.Path)
		return nil
	},
}

var projectRemoveCmd = &cobra.Command{
	Use:               "remove [NAME]",
	Short:             "Remove a Markdown project",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		name := ""
		if len(args) == 1 {
			name = args[0]
		} else {
			selected, err := selectConfiguredProject(store)
			if err != nil || selected == nil {
				return err
			}
			confirmed, err := confirmProjectRemoval(cmd.InOrStdin(), cmd.OutOrStdout(), selected.Name)
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), "Removal cancelled.")
				return nil
			}
			name = selected.Name
		}
		project, err := store.remove(name)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed project %q.\n", project.Name)
		return nil
	},
}

var projectOpenCmd = &cobra.Command{
	Use:               "open [NAME]",
	Short:             "Open a configured Markdown project",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(_ *cobra.Command, args []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		var project markdownProject
		if len(args) == 1 {
			project, err = store.find(args[0])
		} else {
			selected, selectErr := selectConfiguredProject(store)
			if selectErr != nil || selected == nil {
				return selectErr
			}
			project = *selected
		}
		if err != nil {
			return err
		}
		return openProject(project)
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd, projectAddCmd, projectRemoveCmd, projectOpenCmd)
}

func completeProjectNames(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	store, err := configuredProjectStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	projects, err := store.list()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func zeroOrExactArgs(count int) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 || len(args) == count {
			return nil
		}
		return fmt.Errorf("accepts either 0 or %d arg(s), received %d", count, len(args))
	}
}

func addProjectInteractively(cmd *cobra.Command, store projectStore) (markdownProject, error) {
	name, path, err := promptProjectDetails(cmd.InOrStdin(), cmd.OutOrStdout())
	if err != nil {
		return markdownProject{}, err
	}
	return store.add(name, path)
}

func promptProjectDetails(input io.Reader, output io.Writer) (string, string, error) {
	scanner := bufio.NewScanner(input)
	name, err := promptLine(scanner, output, "Project name: ")
	if err != nil {
		return "", "", err
	}
	path, err := promptLine(scanner, output, "Project path: ")
	if err != nil {
		return "", "", err
	}
	return name, path, nil
}

func promptLine(scanner *bufio.Scanner, output io.Writer, label string) (string, error) {
	fmt.Fprint(output, label)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("unable to read input: %w", err)
		}
		return "", errors.New("input closed")
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func selectConfiguredProject(store projectStore) (*markdownProject, error) {
	projects, err := store.list()
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, errors.New("no projects configured; add one with \"glow project add\"")
	}
	return selectProjectForCommand(projects)
}

func confirmProjectRemoval(input io.Reader, output io.Writer, name string) (bool, error) {
	scanner := bufio.NewScanner(input)
	answer, err := promptLine(scanner, output, fmt.Sprintf("Remove project %q? [y/N]: ", name))
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(answer)
	return answer == "y" || answer == "yes", nil
}

func openProject(project markdownProject) error {
	if err := validateProjectDirectory(project.Path); err != nil {
		return err
	}
	return runProjectForCommand(project.Path, "")
}
