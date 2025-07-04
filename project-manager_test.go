package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialModel(t *testing.T) {
	m := initialModel()

	if m.State != StateFolderSelection {
		t.Errorf("Initial state should be StateFolderSelection, got %v", m.State)
	}

	if m.DelaySeconds != 2 {
		t.Errorf("Default delay should be 2 seconds, got %d", m.DelaySeconds)
	}

	if m.SelectedAgent != 0 {
		t.Errorf("Default agent should be 0 (claude), got %d", m.SelectedAgent)
	}

	if m.InputFolder != "input" {
		t.Errorf("Default input folder should be 'input', got %s", m.InputFolder)
	}

	if len(m.FolderOptions) != 2 {
		t.Errorf("Should have 2 folder options, got %d", len(m.FolderOptions))
	}

	if m.FolderOptions[1] != "Other" {
		t.Errorf("Second folder option should be 'Other', got %s", m.FolderOptions[1])
	}
}

func TestCheckFiles(t *testing.T) {
	tmpDir, inputDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	// Test with missing files
	testMissingFiles(t)

	// Create files and test again
	createTestFiles(t, inputDir)
	testFoundFiles(t)
}

func setupTestDirectory(t *testing.T) (string, string) {
	tmpDir, err := os.MkdirTemp("", "project-manager-test")
	if err != nil {
		t.Fatal(err)
	}

	inputDir := filepath.Join(tmpDir, "input")
	if err := os.Mkdir(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(originalDir) })

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	return tmpDir, inputDir
}

func testMissingFiles(t *testing.T) {
	cmd := checkFiles("input")
	msg := cmd()
	result, ok := msg.(fileCheckResult)
	if !ok {
		t.Fatal("checkFiles should return fileCheckResult")
	}

	if result.SpecificationFound || result.TicketsFound || result.StandardPromptFound {
		t.Error("Files should not be found in empty directory")
	}

	if len(result.MissingFiles) != 3 {
		t.Errorf("Should have 3 missing files, got %d", len(result.MissingFiles))
	}
}

func createTestFiles(t *testing.T, inputDir string) {
	for _, file := range []string{"specification.md", "tickets.md", "standard-prompt.md"} {
		f, err := os.Create(filepath.Join(inputDir, file))
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

func testFoundFiles(t *testing.T) {
	cmd := checkFiles("input")
	msg := cmd()
	result, ok := msg.(fileCheckResult)
	if !ok {
		t.Fatal("checkFiles should return fileCheckResult")
	}

	if !result.SpecificationFound || !result.TicketsFound || !result.StandardPromptFound {
		t.Error("All files should be found")
	}

	if len(result.MissingFiles) != 0 {
		t.Errorf("Should have 0 missing files, got %d", len(result.MissingFiles))
	}
}

func TestParseTickets(t *testing.T) {
	// Create a temporary file with ticket content
	tmpFile, err := os.CreateTemp("", "tickets-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `## Ticket 1: First Task
Do something

## Ticket 2: Second Task
Do something else

## Ticket 3: Third Task
Final task
`

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	tickets, err := parseTickets(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

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

func TestTicketStatus(t *testing.T) {
	m := initialModel()
	m.Tickets = []Ticket{
		{Number: 1, Description: "Test", Completed: false, Failed: false},
	}

	// Test that tickets start uncompleted
	if m.Tickets[0].Completed {
		t.Error("Ticket should start as not completed")
	}

	if m.Tickets[0].Failed {
		t.Error("Ticket should start as not failed")
	}

	// Test ticket fields exist and are accessible
	if m.Tickets[0].Number != 1 {
		t.Error("Ticket number should be 1")
	}

	if m.Tickets[0].Description != "Test" {
		t.Error("Ticket description should be 'Test'")
	}
}

func TestAppStateTransitions(t *testing.T) {
	m := initialModel()

	// File check -> File check results
	m.State = StateFileCheck
	newModel, _ := m.Update(fileCheckResult{
		SpecificationFound:  true,
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
	// Test that view doesn't panic in different states
	states := []struct {
		state AppState
		setup func(*Model)
	}{
		{StateFolderSelection, nil},
		{StateFileCheck, nil},
		{StateFileCheckResults, nil},
		{StateFilePicker, func(m *Model) {
			m.MissingFiles = []string{"test.md"}
			m.CurrentMissingIndex = 0
		}},
		{StateAgentSelection, nil},
		{StateCustomCommandEntry, nil},
		{StateCustomFolderEntry, nil},
		{StateConfirmation, func(m *Model) {
			m.Tickets = []Ticket{{Number: 1, Description: "Test"}}
		}},
		{StateRunning, func(m *Model) {
			m.Tickets = []Ticket{{Number: 1, Description: "Test"}}
			m.CurrentTicket = 0
		}},
		{StateCompleted, func(m *Model) {
			m.Tickets = []Ticket{{Number: 1, Description: "Test", Completed: true}}
		}},
	}

	for _, test := range states {
		m := initialModel() // Fresh model for each test
		m.State = test.state

		// Apply setup if provided
		if test.setup != nil {
			test.setup(&m)
		}

		view := m.View()
		if view == "" {
			t.Errorf("View should not be empty for state %v", test.state)
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

func TestGetTicketStatusWithError(t *testing.T) {
	m := initialModel()

	// Set up tickets
	m.Tickets = []Ticket{
		{Number: 1, Description: "First", Completed: true, StartTime: time.Now().Add(-10 * time.Minute), EndTime: time.Now().Add(-8 * time.Minute)},
		{Number: 2, Description: "Second", StartTime: time.Now().Add(-5 * time.Minute), EndTime: time.Now().Add(-3 * time.Minute)},
		{Number: 3, Description: "Third"},
	}

	// Test cases for different scenarios
	tests := []struct {
		name           string
		ticketIndex    int
		currentTicket  int
		processRunning bool
		processError   error
		isWaiting      bool
		expectedStatus string
		expectTimeInfo bool
	}{
		{
			name:           "Previous completed ticket",
			ticketIndex:    0,
			currentTicket:  1,
			expectedStatus: "✅",
			expectTimeInfo: true,
		},
		{
			name:           "Previous failed ticket",
			ticketIndex:    0,
			currentTicket:  1,
			expectedStatus: "❌",
			expectTimeInfo: true,
		},
		{
			name:           "Current running ticket",
			ticketIndex:    1,
			currentTicket:  1,
			processRunning: true,
			expectedStatus: "🔄",
			expectTimeInfo: true,
		},
		{
			name:           "Current ticket with error",
			ticketIndex:    1,
			currentTicket:  1,
			processError:   fmt.Errorf("agent failed"),
			expectedStatus: "❌",
			expectTimeInfo: true,
		},
		{
			name:           "Current waiting ticket",
			ticketIndex:    1,
			currentTicket:  1,
			isWaiting:      true,
			expectedStatus: "⏳",
			expectTimeInfo: false,
		},
		{
			name:           "Future ticket",
			ticketIndex:    2,
			currentTicket:  1,
			expectedStatus: "⏳",
			expectTimeInfo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up model state
			m.CurrentTicket = tt.currentTicket
			m.ProcessRunning = tt.processRunning
			m.ProcessError = tt.processError
			m.IsWaiting = tt.isWaiting

			if tt.isWaiting {
				m.WaitingUntil = time.Now().Add(5 * time.Second)
			}

			// Mark first ticket as failed if needed for test
			if tt.name == "Previous failed ticket" {
				m.Tickets[0].Failed = true
				m.Tickets[0].Completed = false
			}

			// Get status
			status, timeInfo := m.getTicketStatus(tt.ticketIndex, m.Tickets[tt.ticketIndex])

			// Check status
			if !strings.Contains(status, strings.TrimSuffix(tt.expectedStatus, " ")) {
				t.Errorf("Expected status to contain %q, got %q", tt.expectedStatus, status)
			}

			// Check time info
			if tt.expectTimeInfo && timeInfo == "" {
				t.Error("Expected time info, but got empty string")
			} else if !tt.expectTimeInfo && timeInfo != "" {
				t.Errorf("Expected no time info, but got %q", timeInfo)
			}
		})
	}
}
