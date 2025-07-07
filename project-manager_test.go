package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestParseTickets(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedCount   int
		expectedNumbers []int
		expectedDescs   []string
	}{
		{
			name: "Standard double hash format",
			content: `# Project Tickets
## Ticket 1: First task
Some description
## Ticket 2: Second task
## Ticket 3: Third task`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"First task", "Second task", "Third task"},
		},
		{
			name: "Mixed hash levels",
			content: `# Ticket 1: Single hash
## Ticket 2: Double hash
### Ticket 3: Triple hash
#### Ticket 4: Four hashes`,
			expectedCount:   4,
			expectedNumbers: []int{1, 2, 3, 4},
			expectedDescs:   []string{"Single hash", "Double hash", "Triple hash", "Four hashes"},
		},
		{
			name: "Case insensitive",
			content: `# ticket 1: lowercase
## TICKET 2: UPPERCASE
### TiCkEt 3: MixedCase`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"lowercase", "UPPERCASE", "MixedCase"},
		},
		{
			name: "With hash symbol in number",
			content: `## Ticket #1: With hash
### Ticket #2: Another with hash
# Ticket#3: No space before hash`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"With hash", "Another with hash", "No space before hash"},
		},
		{
			name: "Different separators",
			content: `## Ticket 1: Colon separator
### Ticket 2 - Dash separator
# Ticket 3 – Em dash
## Ticket 4 — Long dash`,
			expectedCount:   4,
			expectedNumbers: []int{1, 2, 3, 4},
			expectedDescs:   []string{"Colon separator", "Dash separator", "Em dash", "Long dash"},
		},
		{
			name: "No separator or description",
			content: `## Ticket 1
### Ticket 2
# Ticket 3`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"", "", ""},
		},
		{
			name: "Out of order numbers",
			content: `## Ticket 3: Third
### Ticket 1: First
# Ticket 2: Second`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"First", "Second", "Third"},
		},
		{
			name: "Duplicate numbers (should auto-assign)",
			content: `## Ticket 1: First
### Ticket 1: Duplicate
# Ticket 2: Second`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"First", "Duplicate", "Second"},
		},
		{
			name: "No numbers (should auto-assign)",
			content: `## Ticket: No number
### Ticket: Another no number
# Ticket: Third no number`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"No number", "Another no number", "Third no number"},
		},
		{
			name: "Mixed with non-ticket headers",
			content: `# Project Overview
## Ticket 1: Real ticket
### Some other header
## Ticket 2: Another ticket
# Not a ticket header
### Ticket 3: Third ticket`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"Real ticket", "Another ticket", "Third ticket"},
		},
		{
			name: "Extra spaces",
			content: `##   Ticket   1   :   Extra spaces
###     Ticket 2     -     More spaces
#   Ticket   3`,
			expectedCount:   3,
			expectedNumbers: []int{1, 2, 3},
			expectedDescs:   []string{"Extra spaces", "More spaces", ""},
		},
		{
			name: "Empty file",
			content:         ``,
			expectedCount:   0,
			expectedNumbers: []int{},
			expectedDescs:   []string{},
		},
		{
			name: "Only newlines and spaces",
			content: `


    

`,
			expectedCount:   0,
			expectedNumbers: []int{},
			expectedDescs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpfile, err := ioutil.TempFile("", "test-tickets-*.md")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			// Write test content
			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Parse tickets
			tickets, err := parseTickets(tmpfile.Name())
			if err != nil {
				t.Fatalf("parseTickets() error = %v", err)
			}

			// Check count
			if len(tickets) != tt.expectedCount {
				t.Errorf("parseTickets() returned %d tickets, want %d", len(tickets), tt.expectedCount)
			}

			// Check ticket details
			for i, ticket := range tickets {
				if i >= len(tt.expectedNumbers) {
					break
				}

				if ticket.Number != tt.expectedNumbers[i] {
					t.Errorf("ticket[%d].Number = %d, want %d", i, ticket.Number, tt.expectedNumbers[i])
				}

				if ticket.Description != tt.expectedDescs[i] {
					t.Errorf("ticket[%d].Description = %q, want %q", i, ticket.Description, tt.expectedDescs[i])
				}
			}
		})
	}
}

func TestParseTicketsFileError(t *testing.T) {
	// Test with non-existent file
	_, err := parseTickets("/non/existent/file.md")
	if err == nil {
		t.Error("parseTickets() with non-existent file should return error")
	}
}

func TestCheckFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := ioutil.TempDir("", "test-project-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create input directory
	inputDir := tmpDir + "/input"
	if err := os.Mkdir(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create project directory
	projectDir := inputDir + "/test-project"
	if err := os.Mkdir(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	specContent := "# Specification"
	ticketsContent := `## Ticket 1: First
### Ticket 2: Second`
	promptContent := "Standard prompt"

	if err := ioutil.WriteFile(projectDir+"/specification.md", []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(projectDir+"/tickets.md", []byte(ticketsContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(projectDir+"/standard-prompt.md", []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Test checkFiles
	result := checkFiles("test-project")

	// Verify results
	if !result.SpecificationFound {
		t.Error("specification.md should be found")
	}
	if !result.TicketsFound {
		t.Error("tickets.md should be found")
	}
	if !result.StandardPromptFound {
		t.Error("standard-prompt.md should be found")
	}
	if len(result.MissingFiles) != 0 {
		t.Errorf("No files should be missing, got %v", result.MissingFiles)
	}
	if result.TicketCount != 2 {
		t.Errorf("Should have found 2 tickets, got %d", result.TicketCount)
	}
	if len(result.ParsedTickets) != 2 {
		t.Errorf("Should have parsed 2 tickets, got %d", len(result.ParsedTickets))
	}
}

func TestCheckFilesMissing(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := ioutil.TempDir("", "test-project-missing-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create input directory
	inputDir := tmpDir + "/input"
	if err := os.Mkdir(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create project directory
	projectDir := inputDir + "/test-project"
	if err := os.Mkdir(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Only create specification.md
	if err := ioutil.WriteFile(projectDir+"/specification.md", []byte("# Spec"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Test checkFiles
	result := checkFiles("test-project")

	// Verify results
	if !result.SpecificationFound {
		t.Error("specification.md should be found")
	}
	if result.TicketsFound {
		t.Error("tickets.md should not be found")
	}
	if result.StandardPromptFound {
		t.Error("standard-prompt.md should not be found")
	}
	if len(result.MissingFiles) != 2 {
		t.Errorf("Should have 2 missing files, got %d", len(result.MissingFiles))
	}
	if result.TicketCount != 0 {
		t.Errorf("Should have found 0 tickets, got %d", result.TicketCount)
	}
}

func BenchmarkParseTickets(b *testing.B) {
	// Create a test file with many tickets
	content := ""
	for i := 1; i <= 100; i++ {
		content += fmt.Sprintf("## Ticket %d: Task number %d\n", i, i)
		content += "Some description for this ticket\n\n"
	}

	tmpfile, err := ioutil.TempFile("", "bench-tickets-*.md")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		b.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseTickets(tmpfile.Name())
	}
}