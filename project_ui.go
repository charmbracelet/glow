package main

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type projectItem struct {
	project markdownProject
}

func (i projectItem) Title() string       { return i.project.Name }
func (i projectItem) Description() string { return i.project.Path }
func (i projectItem) FilterValue() string { return i.project.Name + " " + i.project.Path }

type projectPickerModel struct {
	list     list.Model
	selected *markdownProject
}

func newProjectPickerModel(projects []markdownProject) projectPickerModel {
	items := make([]list.Item, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectItem{project: project})
	}
	projectList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	projectList.KeyMap.CursorUp.SetKeys("up", "k", "ctrl+p")
	projectList.KeyMap.CursorDown.SetKeys("down", "j", "ctrl+n")
	projectList.Title = "Glow Projects"
	projectList.SetShowStatusBar(true)
	projectList.SetFilteringEnabled(true)
	return projectPickerModel{list: projectList}
}

func (m projectPickerModel) Init() tea.Cmd {
	return nil
}

func (m projectPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "enter" && m.list.FilterState() != list.Filtering {
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				selected := item.project
				m.selected = &selected
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m projectPickerModel) View() tea.View {
	view := tea.NewView(m.list.View())
	view.AltScreen = true
	return view
}

func selectProject(projects []markdownProject) (*markdownProject, error) {
	program := tea.NewProgram(newProjectPickerModel(projects))
	result, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to run project selector: %w", err)
	}
	model, ok := result.(projectPickerModel)
	if !ok {
		return nil, fmt.Errorf("project selector returned %T", result)
	}
	return model.selected, nil
}
