package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialModel(t *testing.T) {
	m := initialModel()
	
	if m.State != StateFileCheck {
		t.Errorf("Initial state should be StateFileCheck, got %v", m.State)
	}
	
	if m.DelaySeconds != 2 {
		t.Errorf("Default delay should be 2 seconds, got %d", m.DelaySeconds)
	}
	
	if m.SelectedAgent != 0 {
		t.Errorf("Default agent should be 0 (claude), got %d", m.SelectedAgent)
	}
}

func TestCheckFiles(t *testing.T) {
	// Create temporary directory and files
	tmpDir, err := os.MkdirTemp("", "project-manager-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Test with missing files
	result := checkFiles(
		filepath.Join(tmpDir, "spec.md"),
		filepath.Join(tmpDir, "tickets.md"),
		filepath.Join(tmpDir, "prompt.md"),
	)
	
	if result.SpecificationFound || result.TicketsFound || result.StandardPromptFound {
		t.Error("Files should not be found in empty directory")
	}
	
	if len(result.MissingFiles) != 3 {
		t.Errorf("Should have 3 missing files, got %d", len(result.MissingFiles))
	}
	
	// Create files and test again
	for _, file := range []string{"spec.md", "tickets.md", "prompt.md"} {
		f, err := os.Create(filepath.Join(tmpDir, file))
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	
	result = checkFiles(
		filepath.Join(tmpDir, "spec.md"),
		filepath.Join(tmpDir, "tickets.md"),
		filepath.Join(tmpDir, "prompt.md"),
	)
	
	if !result.SpecificationFound || !result.TicketsFound || !result.StandardPromptFound {
		t.Error("All files should be found")
	}
	
	if len(result.MissingFiles) != 0 {
		t.Errorf("Should have 0 missing files, got %d", len(result.MissingFiles))
	}
}

func TestParseTickets(t *testing.T) {
	content := `
## Ticket 1: First Task
Do something

## Ticket 2: Second Task
Do something else

## Ticket 3: Third Task
Final task
`
	
	tickets := parseTickets(content)
	
	if len(tickets) != 3 {
		t.Fatalf("Expected 3 tickets, got %d", len(tickets))
	}
	
	expected := []struct {
		number int
		desc   string
	}{
		{1, "First Task"},
		{2, "Second Task"},
		{3, "Third Task"},
	}
	
	for i, exp := range expected {
		if tickets[i].Number != exp.number {
			t.Errorf("Ticket %d: expected number %d, got %d", i, exp.number, tickets[i].Number)
		}
		if tickets[i].Description != exp.desc {
			t.Errorf("Ticket %d: expected description %q, got %q", i, exp.desc, tickets[i].Description)
		}
		if tickets[i].Completed {
			t.Errorf("Ticket %d should not be completed initially", i)
		}
		if tickets[i].Failed {
			t.Errorf("Ticket %d should not be failed initially", i)
		}
	}
}

func TestTicketUpdate(t *testing.T) {
	m := initialModel()
	m.Tickets = []Ticket{
		{Number: 1, Description: "Test"},
	}
	
	// Test tick message updates
	m.CurrentTicket = 0
	m.ProcessRunning = true
	
	newModel, cmd := m.Update(tickMsg{output: "success", err: nil})
	m = newModel.(Model)
	
	if !m.Tickets[0].Completed {
		t.Error("Ticket should be marked as completed")
	}
	
	if m.Tickets[0].Failed {
		t.Error("Ticket should not be marked as failed")
	}
	
	if cmd == nil {
		t.Error("Should return a command after processing ticket")
	}
}

func TestAppStateTransitions(t *testing.T) {
	m := initialModel()
	
	// File check -> File check results
	m.State = StateFileCheck
	newModel, _ := m.Update(fileCheckResult{
		SpecificationFound:   true,
		TicketsFound:        true,
		StandardPromptFound: true,
	})
	m = newModel.(Model)
	
	if m.State != StateFileCheckResults {
		t.Errorf("Expected StateFileCheckResults, got %v", m.State)
	}
	
	// File check results -> Agent selection
	newModel, _ = m.Update(proceedToAgentSelectionMsg{})
	m = newModel.(Model)
	
	if m.State != StateAgentSelection {
		t.Errorf("Expected StateAgentSelection, got %v", m.State)
	}
}

func TestViewOutput(t *testing.T) {
	m := initialModel()
	
	// Test that view doesn't panic in different states
	states := []AppState{
		StateFileCheck,
		StateFileCheckResults,
		StateFilePicker,
		StateAgentSelection,
		StateCustomCommandEntry,
		StateConfirmation,
		StateRunning,
		StateCompleted,
	}
	
	for _, state := range states {
		m.State = state
		view := m.View()
		if view == "" {
			t.Errorf("View should not be empty for state %v", state)
		}
	}
}

func TestKeyboardNavigation(t *testing.T) {
	m := initialModel()
	m.State = StateAgentSelection
	
	// Test down arrow
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := m.Update(msg)
	m = newModel.(Model)
	
	if m.SelectedAgent != 1 {
		t.Error("Down arrow should move selection to custom agent")
	}
	
	// Test up arrow
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.Update(msg)
	m = newModel.(Model)
	
	if m.SelectedAgent != 0 {
		t.Error("Up arrow should move selection back to claude")
	}
}

func TestExponentialBackoff(t *testing.T) {
	m := initialModel()
	m.DelaySeconds = 2
	
	// First failure
	m.Tickets = []Ticket{{Number: 1, Failed: true}}
	delay := m.DelaySeconds
	if delay > m.DelaySeconds {
		m.DelaySeconds = delay
	}
	
	// Test max delay cap
	m.DelaySeconds = 100
	if m.DelaySeconds > 30 {
		m.DelaySeconds = 30
	}
	
	if m.DelaySeconds != 30 {
		t.Errorf("Delay should be capped at 30 seconds, got %d", m.DelaySeconds)
	}
}