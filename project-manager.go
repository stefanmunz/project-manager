package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("46"))

	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)
)

type tickMsg struct {
	output string
	err    error
}

type processCompleteMsg struct{}

type waitingDoneMsg struct{}

type checkKillFileMsg struct{}

type killFileFoundMsg struct {
	content string
}

type processStartedMsg struct {
	cmd *exec.Cmd
}

type fileCheckResult struct {
	SpecificationFound   bool
	TicketsFound        bool
	StandardPromptFound bool
	MissingFiles        []string
	TicketCount         int
	ParsedTickets       []Ticket
}

type proceedToAgentSelectionMsg struct{}

type projectsScannedMsg struct {
	projects []string
}

type AppState int

const (
	StateProjectSelection AppState = iota
	StateFileCheck
	StateFileCheckResults
	StateFilePicker
	StateAgentSelection
	StateCustomCommandEntry
	StateConfirmation
	StateRunning
	StateCompleted
)

type Model struct {
	State           AppState
	Width           int
	Height          int
	
	// Project selection
	AvailableProjects   []string
	SelectedProject     string
	SelectedProjectIndex int
	
	// File paths
	SpecificationPath   string
	TicketsPath         string
	StandardPromptPath  string
	MissingFiles        []string
	CurrentMissingIndex int
	
	// Components
	FilePicker filepicker.Model
	TextInput  textinput.Model
	
	// Agent selection
	SelectedAgent      int // 0 for claude-code, 1 for other
	CustomAgentCommand string
	
	// Execution state
	Tickets          []Ticket
	CurrentTicket    int
	ProcessRunning   bool
	ProcessError     error
	CurrentCmd       *exec.Cmd // Track running command
	DelaySeconds     int       // Delay between agents
	IsWaiting        bool      // Whether we're in waiting state
	WaitingUntil     time.Time // When to start next agent
	
	// UI state
	Cursor        int
	ConfirmReady  bool
}

type Ticket struct {
	Number      int
	Description string
	Completed   bool
	Failed      bool
	StartTime   time.Time
	EndTime     time.Time
}

func initialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter custom agent command..."
	ti.CharLimit = 200
	
	return Model{
		State:              StateProjectSelection,
		MissingFiles:       []string{},
		TextInput:          ti,
		SelectedAgent:      0,
		Tickets:            []Ticket{},
		DelaySeconds:       2, // Default 2 second delay between agents
		SelectedProjectIndex: 0,
		AvailableProjects:   []string{},
	}
}

func (m Model) Init() tea.Cmd {
	return scanProjects
}

func scanProjects() tea.Msg {
	projects := []string{}
	
	files, err := ioutil.ReadDir("input")
	if err != nil {
		return projectsScannedMsg{projects: projects}
	}
	
	for _, file := range files {
		if file.IsDir() {
			projects = append(projects, file.Name())
		}
	}
	
	return projectsScannedMsg{projects: projects}
}

func checkFilesCmd(project string) tea.Cmd {
	return func() tea.Msg {
		return checkFiles(project)
	}
}

func checkFiles(project string) fileCheckResult {
	result := fileCheckResult{
		MissingFiles:  []string{},
		ParsedTickets: []Ticket{},
	}
	
	specPath := fmt.Sprintf("input/%s/specification.md", project)
	ticketsPath := fmt.Sprintf("input/%s/tickets.md", project)
	promptPath := fmt.Sprintf("input/%s/standard-prompt.md", project)
	
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		result.MissingFiles = append(result.MissingFiles, "specification.md")
	} else {
		result.SpecificationFound = true
	}
	
	if _, err := os.Stat(ticketsPath); os.IsNotExist(err) {
		result.MissingFiles = append(result.MissingFiles, "tickets.md")
	} else {
		result.TicketsFound = true
		// Parse tickets to get count
		if tickets, err := parseTickets(ticketsPath); err == nil {
			result.ParsedTickets = tickets
			result.TicketCount = len(tickets)
		}
	}
	
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		result.MissingFiles = append(result.MissingFiles, "standard-prompt.md")
	} else {
		result.StandardPromptFound = true
	}
	
	return result
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.CurrentCmd != nil && m.CurrentCmd.Process != nil {
				m.CurrentCmd.Process.Kill()
			}
			return m, tea.Quit
			
		case "up", "k":
			if m.State == StateProjectSelection {
				if m.SelectedProjectIndex > 0 {
					m.SelectedProjectIndex--
				}
			} else if m.State == StateAgentSelection {
				m.SelectedAgent = (m.SelectedAgent - 1 + 2) % 2
			}
			
		case "down", "j":
			if m.State == StateProjectSelection {
				if m.SelectedProjectIndex < len(m.AvailableProjects)-1 {
					m.SelectedProjectIndex++
				}
			} else if m.State == StateAgentSelection {
				m.SelectedAgent = (m.SelectedAgent + 1) % 2
			}
		
		case "enter":
			switch m.State {
			case StateProjectSelection:
				if len(m.AvailableProjects) > 0 && m.SelectedProjectIndex < len(m.AvailableProjects) {
					m.SelectedProject = m.AvailableProjects[m.SelectedProjectIndex]
					// Update file paths based on selected project
					m.SpecificationPath = fmt.Sprintf("input/%s/specification.md", m.SelectedProject)
					m.TicketsPath = fmt.Sprintf("input/%s/tickets.md", m.SelectedProject)
					m.StandardPromptPath = fmt.Sprintf("input/%s/standard-prompt.md", m.SelectedProject)
					m.State = StateFileCheck
					return m, checkFilesCmd(m.SelectedProject)
				}
			
			case StateAgentSelection:
				if m.SelectedAgent == 0 {
					m.CustomAgentCommand = "claude --dangerously-skip-permissions"
					m.State = StateConfirmation
				} else {
					// Move to custom command entry state
					m.State = StateCustomCommandEntry
					m.TextInput.SetValue("") // Clear any previous value
					m.TextInput.Focus()
					return m, m.TextInput.Focus()
				}
			
			case StateCustomCommandEntry:
				if m.TextInput.Value() != "" {
					m.CustomAgentCommand = m.TextInput.Value()
					m.State = StateConfirmation
				}
			
			case StateConfirmation:
				m.ConfirmReady = true
			}
		}
		
		// Handle text input in custom command entry
		if m.State == StateCustomCommandEntry {
			var cmd tea.Cmd
			m.TextInput, cmd = m.TextInput.Update(msg)
			return m, cmd
		}
		
		// Handle any key press in StateFileCheckResults
		if m.State == StateFileCheckResults {
			// Any key press moves to agent selection
			return m.Update(proceedToAgentSelectionMsg{})
		}
		
		// Handle file picker navigation
		if m.State == StateFilePicker {
			var cmd tea.Cmd
			m.FilePicker, cmd = m.FilePicker.Update(msg)
			
			if didSelect, path := m.FilePicker.DidSelectFile(msg); didSelect {
				switch m.MissingFiles[m.CurrentMissingIndex] {
				case "specification.md":
					m.SpecificationPath = path
				case "tickets.md":
					m.TicketsPath = path
				case "standard-prompt.md":
					m.StandardPromptPath = path
				}
				
				m.CurrentMissingIndex++
				if m.CurrentMissingIndex >= len(m.MissingFiles) {
					m.State = StateAgentSelection
				} else {
					fp := filepicker.New()
					fp.CurrentDirectory, _ = os.Getwd()
					m.FilePicker = fp
					return m, m.FilePicker.Init()
				}
			}
			return m, cmd
		}

	case tickMsg:
		if msg.err != nil {
			m.ProcessError = msg.err
			m.ProcessRunning = false
			
			// Check if it's an API overload error and increase delay
			if strings.Contains(msg.output, "server overload") || 
			   strings.Contains(msg.output, "rate limit") ||
			   strings.Contains(msg.output, "too many requests") {
				// Exponential backoff: double the delay, max 30 seconds
				m.DelaySeconds = m.DelaySeconds * 2
				if m.DelaySeconds > 30 {
					m.DelaySeconds = 30
				}
			}
			// Continue to next agent even on error (after delay)
			return m.Update(processCompleteMsg{})
		}
		// If we got output but no error, the process succeeded
		// Reset delay to default on success
		m.DelaySeconds = 2
		return m.Update(processCompleteMsg{})

	case processStartedMsg:
		// Store the running command
		m.CurrentCmd = msg.cmd
		// Record start time for this ticket
		m.Tickets[m.CurrentTicket].StartTime = time.Now()
		// Start monitoring for kill file and update timer
		return m, tea.Batch(
			checkForKillFile(),
			// Update the view every second to show live duration
			tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return t
			}),
		)

	case checkKillFileMsg:
		// Check if killmenow.md exists
		if content, err := ioutil.ReadFile("killmenow.md"); err == nil {
			// File found, return the content
			return m.Update(killFileFoundMsg{content: string(content)})
		}
		// File not found, check again in 500ms
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return checkKillFileMsg{}
		})

	case killFileFoundMsg:
		// Kill the process
		if m.CurrentCmd != nil && m.CurrentCmd.Process != nil {
			m.CurrentCmd.Process.Kill()
			m.CurrentCmd = nil
		}
		
		// Delete the kill file
		os.Remove("killmenow.md")
		
		// Determine success or failure
		if strings.Contains(strings.ToLower(msg.content), "success") {
			m.ProcessError = nil
		} else {
			m.ProcessError = fmt.Errorf("agent reported failure")
		}
		
		// Move to completion
		return m.Update(processCompleteMsg{})

	case processCompleteMsg:
		m.ProcessRunning = false
		
		// Record end time for this ticket
		m.Tickets[m.CurrentTicket].EndTime = time.Now()
		
		// Mark ticket as completed or failed based on error state
		if m.ProcessError != nil {
			m.Tickets[m.CurrentTicket].Failed = true
		} else {
			m.Tickets[m.CurrentTicket].Completed = true
		}
		
		m.CurrentTicket++
		
		if m.CurrentTicket < len(m.Tickets) {
			// Don't clear output yet - keep it visible during waiting
			// We'll clear it when we actually start the next agent
			
			// Start waiting period
			m.IsWaiting = true
			m.WaitingUntil = time.Now().Add(time.Duration(m.DelaySeconds) * time.Second)
			m.ProcessRunning = false
			
			// Return commands for both the waiting timer and the countdown update
			return m, tea.Batch(
				tea.Tick(time.Duration(m.DelaySeconds)*time.Second, func(t time.Time) tea.Msg {
					return waitingDoneMsg{}
				}),
				tea.Tick(time.Second, func(t time.Time) tea.Msg {
					return t
				}),
			)
		} else {
			m.State = StateCompleted
		}
		return m, nil
	
	case waitingDoneMsg:
		// Waiting period is over, start next agent
		m.IsWaiting = false
		
		// Clear error state for next agent
		m.ProcessError = nil
		m.ProcessRunning = true
		return m, m.runNextAgent()
	
	case time.Time:
		// Update the view to refresh the countdown or running time
		if m.IsWaiting && time.Now().Before(m.WaitingUntil) {
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return t
			})
		} else if m.ProcessRunning && m.State == StateRunning {
			// Continue updating while a process is running to show live duration
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return t
			})
		}
		return m, nil

	case projectsScannedMsg:
		m.AvailableProjects = msg.projects
		if len(m.AvailableProjects) == 0 {
			// No projects found - we could show an error
			m.ProcessError = fmt.Errorf("No project folders found in input/")
			m.State = StateCompleted
		}
		return m, nil

	case fileCheckResult:
		// File check results
		// Store the parsed tickets from file check
		if len(msg.ParsedTickets) > 0 {
			m.Tickets = msg.ParsedTickets
		}
		
		if len(msg.MissingFiles) == 0 {
			// All files found - show results and wait for user input
			m.State = StateFileCheckResults
			return m, nil
		} else {
			// Some files missing
			m.MissingFiles = msg.MissingFiles
			m.State = StateFilePicker
			m.CurrentMissingIndex = 0
			
			fp := filepicker.New()
			fp.CurrentDirectory, _ = os.Getwd()
			m.FilePicker = fp
			return m, m.FilePicker.Init()
		}
	
	case proceedToAgentSelectionMsg:
		// Transition from file check results to agent selection
		if m.State == StateFileCheckResults {
			// Tickets are already parsed during file check, no need to re-parse
			m.State = StateAgentSelection
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	// Update states
	switch m.State {
	case StateConfirmation:
		if m.ConfirmReady {
			m.State = StateRunning
			m.ProcessRunning = true
			m.CurrentTicket = 0
			return m, m.runNextAgent()
		}
		m.ConfirmReady = true
	}
	
	return m, nil
}

func parseTickets(path string) ([]Ticket, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	// Create regex patterns for different ticket formats
	// Matches: # Ticket 1, ## Ticket 2:, ### ticket 3 -, #### TICKET #4, etc.
	// Also matches tickets without numbers: ## Ticket: Description
	ticketRegex := regexp.MustCompile(`(?i)^#{1,}\s*ticket\s*(?:#?\s*(\d+))?\s*[:|\-–—]?\s*(.*)`)
	
	lines := strings.Split(string(content), "\n")
	tickets := []Ticket{}
	ticketMap := make(map[int]bool) // To avoid duplicate ticket numbers
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := ticketRegex.FindStringSubmatch(line)
		
		if len(matches) > 0 {
			// Extract ticket number (capture group 1)
			ticketNum := 0
			if len(matches) > 1 && matches[1] != "" {
				fmt.Sscanf(matches[1], "%d", &ticketNum)
			}
			
			// If no number found or already exists, auto-assign
			if ticketNum == 0 || ticketMap[ticketNum] {
				ticketNum = len(tickets) + 1
			}
			ticketMap[ticketNum] = true
			
			// Extract description (capture group 2)
			desc := ""
			if len(matches) > 2 {
				desc = strings.TrimSpace(matches[2])
			}
			
			tickets = append(tickets, Ticket{
				Number:      ticketNum,
				Description: desc,
			})
		}
	}
	
	// Sort tickets by number (in case they were out of order)
	// Simple bubble sort for small lists
	for i := 0; i < len(tickets); i++ {
		for j := i + 1; j < len(tickets); j++ {
			if tickets[i].Number > tickets[j].Number {
				tickets[i], tickets[j] = tickets[j], tickets[i]
			}
		}
	}
	
	return tickets, nil
}

func (m Model) runNextAgent() tea.Cmd {
	return func() tea.Msg {
		standardPrompt, err := ioutil.ReadFile(m.StandardPromptPath)
		if err != nil {
			return tickMsg{output: "", err: err}
		}
		
		// Add kill file instruction to the prompt
		prompt := fmt.Sprintf("%s Please use the documentation in the input/%s folder, especially the specification.md and the tickets.md. Please work on ticket %d. As your final task, create a file named 'killmenow.md' containing either 'success' or 'failure' to indicate whether you successfully completed the task.",
			string(standardPrompt), m.SelectedProject, m.CurrentTicket+1)
		
		cmdParts := strings.Fields(m.CustomAgentCommand)
		if len(cmdParts) == 0 {
			return tickMsg{output: "", err: fmt.Errorf("invalid command")}
		}
		
		// Append prompt as a command-line argument
		args := append(cmdParts[1:], prompt)
		cmd := exec.Command(cmdParts[0], args...)
		
		// Start the command asynchronously
		if err := cmd.Start(); err != nil {
			return tickMsg{output: "", err: err}
		}
		
		// Return a message indicating the process has started
		return processStartedMsg{cmd: cmd}
	}
}

func checkForKillFile() tea.Cmd {
	return func() tea.Msg {
		return checkKillFileMsg{}
	}
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%d hour%s, %d minute%s, %d second%s", 
			hours, plural(hours), minutes, plural(minutes), seconds, plural(seconds))
	} else if minutes > 0 {
		return fmt.Sprintf("%d minute%s, %d second%s", 
			minutes, plural(minutes), seconds, plural(seconds))
	} else {
		return fmt.Sprintf("%d second%s", seconds, plural(seconds))
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func (m Model) View() string {
	s := titleStyle.Render("Project Manager") + "\n\n"
	
	switch m.State {
	case StateProjectSelection:
		s += "Select a project to work on:\n\n"
		
		if len(m.AvailableProjects) == 0 {
			s += errorStyle.Render("No project folders found in input/") + "\n"
			s += infoStyle.Render("Please create a project folder with specification.md, tickets.md, and standard-prompt.md files")
		} else {
			for i, project := range m.AvailableProjects {
				if i == m.SelectedProjectIndex {
					s += selectedStyle.Render("→ " + project) + "\n"
				} else {
					s += "  " + project + "\n"
				}
			}
			s += "\n" + infoStyle.Render("Use ↑/↓ to navigate, Enter to select")
		}
		
	case StateFileCheck:
		s += "Checking for required files...\n"
		
	case StateFileCheckResults:
		s += "Checking for required files...\n\n"
		s += successStyle.Render("✅ Successfully found specification.md") + "\n"
		s += successStyle.Render(fmt.Sprintf("✅ Successfully found tickets.md (%d tickets)", len(m.Tickets))) + "\n"
		s += successStyle.Render("✅ Successfully found standard-prompt.md") + "\n\n"
		s += infoStyle.Render("All files found! Press any key to continue...")
		
	case StateFilePicker:
		s += fmt.Sprintf("Missing file: %s\n", errorStyle.Render(m.MissingFiles[m.CurrentMissingIndex]))
		s += "Please select the file location:\n\n"
		s += m.FilePicker.View()
		
	case StateAgentSelection:
		s += "Select coding agent:\n\n"
		
		choices := []string{
			"claude --dangerously-skip-permissions",
			"Other (enter custom command)",
		}
		
		for i, choice := range choices {
			if i == m.SelectedAgent {
				s += selectedStyle.Render("→ " + choice) + "\n"
			} else {
				s += "  " + choice + "\n"
			}
		}
		
		s += "\n" + infoStyle.Render("Press Enter to continue")
		
	case StateCustomCommandEntry:
		s += "Enter custom agent command:\n\n"
		s += m.TextInput.View() + "\n\n"
		s += infoStyle.Render("Press Enter when done")
		
	case StateConfirmation:
		s += "Ready to start execution:\n\n"
		s += fmt.Sprintf("📂 Project: %s\n", m.SelectedProject)
		s += fmt.Sprintf("📁 Specification: %s\n", m.SpecificationPath)
		s += fmt.Sprintf("📋 Tickets: %s (%d tickets)\n", m.TicketsPath, len(m.Tickets))
		s += fmt.Sprintf("📝 Prompt: %s\n", m.StandardPromptPath)
		s += fmt.Sprintf("🤖 Agent: %s\n", m.CustomAgentCommand)
		s += fmt.Sprintf("⏱️  Delay between agents: %d seconds\n", m.DelaySeconds)
		
		s += "\n" + successStyle.Render("Press Enter to start")
		
	case StateRunning:
		s += fmt.Sprintf("Executing agents... (Ticket %d/%d)\n\n", m.CurrentTicket+1, len(m.Tickets))
		
		// Show ticket status with emojis
		for i, ticket := range m.Tickets {
			var status string
			var timeInfo string
			
			if i < m.CurrentTicket {
				// Completed tickets - show duration
				if ticket.Failed {
					status = "❌"
				} else {
					status = "✅"
				}
				duration := ticket.EndTime.Sub(ticket.StartTime)
				timeInfo = fmt.Sprintf(" - %s", formatDuration(duration))
			} else if i == m.CurrentTicket {
				// Current ticket
				if m.ProcessRunning {
					status = "🔄"
					// Show live duration for running ticket
					if !ticket.StartTime.IsZero() {
						currentDuration := time.Since(ticket.StartTime)
						timeInfo = fmt.Sprintf(" - %s", formatDuration(currentDuration))
					}
				} else if m.IsWaiting {
					remainingTime := int(m.WaitingUntil.Sub(time.Now()).Seconds())
					if remainingTime < 0 {
						remainingTime = 0
					}
					status = fmt.Sprintf("⏳ (%ds)", remainingTime)
				} else {
					status = "⏸️"
				}
			} else {
				status = "⏳"
			}
			
			s += fmt.Sprintf("%s Ticket %d: %s%s\n", status, ticket.Number, ticket.Description, timeInfo)
		}
		
		if m.ProcessError != nil {
			s += "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.ProcessError)) + "\n"
		}
		
	case StateCompleted:
		s += successStyle.Render("All agents completed!") + "\n\n"
		
		// Show detailed ticket results with timing
		var totalDuration time.Duration
		successful := 0
		failed := 0
		
		for _, ticket := range m.Tickets {
			duration := ticket.EndTime.Sub(ticket.StartTime)
			totalDuration += duration
			
			status := "✅"
			if ticket.Failed {
				status = "❌"
				failed++
			} else {
				successful++
			}
			
			s += fmt.Sprintf("%s Ticket %d: %s - %s\n", 
				status, ticket.Number, ticket.Description, formatDuration(duration))
		}
		
		// Show summary
		s += fmt.Sprintf("\nSummary:\n")
		s += fmt.Sprintf("✅ Successful: %d\n", successful)
		s += fmt.Sprintf("❌ Failed: %d\n", failed)
		s += fmt.Sprintf("📊 Total: %d\n", len(m.Tickets))
		s += fmt.Sprintf("⏱️  Total time: %s\n", formatDuration(totalDuration))
		
		s += "\n" + infoStyle.Render("Press q to quit")
	}
	
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}