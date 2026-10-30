package main

import (
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func TestProjectPickerSelectsProject(t *testing.T) {
	projects := []markdownProject{
		{Name: "First", Path: "/first"},
		{Name: "Second", Path: "/second"},
	}
	model := newProjectPickerModel(projects)

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(projectPickerModel)
	if cmd == nil {
		t.Fatal("expected selecting a project to quit the picker")
	}
	if model.selected == nil || *model.selected != projects[0] {
		t.Fatalf("expected first project to be selected, got %#v", model.selected)
	}
}

func TestProjectPickerControlKeysMoveSelection(t *testing.T) {
	projects := []markdownProject{
		{Name: "First", Path: "/first"},
		{Name: "Second", Path: "/second"},
		{Name: "Third", Path: "/third"},
	}
	model := newProjectPickerModel(projects)

	updated, _ := model.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	model = updated.(projectPickerModel)
	if got := model.list.Index(); got != 1 {
		t.Fatalf("expected ctrl+n to move selection down to 1, got %d", got)
	}

	updated, _ = model.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	model = updated.(projectPickerModel)
	if got := model.list.Index(); got != 0 {
		t.Fatalf("expected ctrl+p to move selection up to 0, got %d", got)
	}
}

func TestProjectPickerControlKeysDoNotApplyFilter(t *testing.T) {
	model := newProjectPickerModel([]markdownProject{
		{Name: "First", Path: "/first"},
		{Name: "Second", Path: "/second"},
	})

	updated, _ := model.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	model = updated.(projectPickerModel)
	updated, _ = model.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	model = updated.(projectPickerModel)

	if model.list.FilterState() != list.Filtering {
		t.Fatalf("expected ctrl+n to leave filter input active, got %s", model.list.FilterState())
	}
}

func TestProjectPickerDoesNotSelectWhileFiltering(t *testing.T) {
	model := newProjectPickerModel([]markdownProject{{Name: "First", Path: "/first"}})

	updated, _ := model.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	model = updated.(projectPickerModel)
	if model.list.FilterState() != list.Filtering {
		t.Fatalf("expected picker to enter filtering state, got %s", model.list.FilterState())
	}

	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updated.(projectPickerModel)
	if model.selected != nil {
		t.Fatalf("expected enter to apply the filter without selecting, got %#v", model.selected)
	}
}

func TestProjectPickerCanBeCancelled(t *testing.T) {
	model := newProjectPickerModel([]markdownProject{{Name: "First", Path: "/first"}})

	updated, cmd := model.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	model = updated.(projectPickerModel)
	if cmd == nil {
		t.Fatal("expected q to quit the picker")
	}
	if model.selected != nil {
		t.Fatalf("expected cancelled picker to have no selection, got %#v", model.selected)
	}
}

func TestProjectPickerItemIncludesNameAndPath(t *testing.T) {
	item := projectItem{project: markdownProject{Name: "Docs", Path: "/docs"}}
	if item.Title() != "Docs" || item.Description() != "/docs" || item.FilterValue() != "Docs /docs" {
		t.Fatalf("unexpected project item values: %#v", item)
	}
}
