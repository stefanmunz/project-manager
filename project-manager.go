package main

import (
	"fmt"
	"os"
	"os/exec"
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
	SpecificationFound  bool
	TicketsFound        bool
	StandardPromptFound bool
	MissingFiles        []string
}

type proceedToAgentSelectionMsg struct{}

// AppState represents the current state of the application
type AppState int

const (
	StateFileCheck AppState = iota
	StateFileCheckResults
	StateFilePicker
	StateAgentSelection
	StateCustomCommandEntry
	StateConfirmation
	StateRunning
	StateCompleted
)

// Model represents the application's state and implements the tea.Model interface
type Model struct {
	State  AppState
	Width  int
	Height int

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
	Tickets        []Ticket
	CurrentTicket  int
	ProcessRunning bool
	ProcessError   error
	CurrentCmd     *exec.Cmd // Track running command
	DelaySeconds   int       // Delay between agents
	IsWaiting      bool      // Whether we're in waiting state
	WaitingUntil   time.Time // When to start next agent

	// UI state
	Cursor       int
	ConfirmReady bool
}

// Ticket represents a single task to be executed by an agent
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
		State:              StateFileCheck,
		SpecificationPath:  "input/specification.md",
		TicketsPath:        "input/tickets.md",
		StandardPromptPath: "input/standard-prompt.md",
		MissingFiles:       []string{},
		TextInput:          ti,
		SelectedAgent:      0,
		Tickets:            []Ticket{},
		DelaySeconds:       2, // Default 2 second delay between agents
	}
}

// Init initializes the model and returns the initial command to check files
func (m Model) Init() tea.Cmd {
	return checkFiles
}

func checkFiles() tea.Msg {
	m := initialModel()
	result := fileCheckResult{
		MissingFiles: []string{},
	}

	if _, err := os.Stat(m.SpecificationPath); os.IsNotExist(err) {
		result.MissingFiles = append(result.MissingFiles, "specification.md")
	} else {
		result.SpecificationFound = true
	}

	if _, err := os.Stat(m.TicketsPath); os.IsNotExist(err) {
		result.MissingFiles = append(result.MissingFiles, "tickets.md")
	} else {
		result.TicketsFound = true
	}

	if _, err := os.Stat(m.StandardPromptPath); os.IsNotExist(err) {
		result.MissingFiles = append(result.MissingFiles, "standard-prompt.md")
	} else {
		result.StandardPromptFound = true
	}

	return result
}

// Update handles incoming messages and updates the model state accordingly
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.CurrentCmd != nil && m.CurrentCmd.Process != nil {
				_ = m.CurrentCmd.Process.Kill()
			}
			return m, tea.Quit

		case "up", "k":
			if m.State == StateAgentSelection {
				m.SelectedAgent = (m.SelectedAgent - 1 + 2) % 2
			}

		case "down", "j":
			if m.State == StateAgentSelection {
				m.SelectedAgent = (m.SelectedAgent + 1) % 2
			}

		case "enter":
			switch m.State {
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
				m.DelaySeconds *= 2
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
		if content, err := os.ReadFile("killmenow.md"); err == nil {
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
			_ = m.CurrentCmd.Process.Kill()
			m.CurrentCmd = nil
		}

		// Log the completion
		now := time.Now()
		logFileName := fmt.Sprintf("%s-%s-party-agent-%d.log",
			strings.ToLower(now.Format("Monday")),
			now.Format("15-04-05"),
			m.CurrentTicket+1)
		
		if logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(logFile, "\n--- Agent Completed at %s ---\n", now.Format("15:04:05"))
			fmt.Fprintf(logFile, "Kill file content: %s\n", msg.content)
			fmt.Fprintf(logFile, "Party.sh exists: %v\n", fileExists("party.sh"))
			if content, err := os.ReadFile("party.sh"); err == nil {
				fmt.Fprintf(logFile, "Party.sh size: %d bytes\n", len(content))
			}
			logFile.Close()
		}

		// Delete the kill file
		_ = os.Remove("killmenow.md")

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

	case fileCheckResult:
		// File check results
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
			tickets, err := parseTickets(m.TicketsPath)
			if err != nil {
				// Handle error silently or set an error state if needed
			} else {
				m.Tickets = tickets
			}
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
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	tickets := []Ticket{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## Ticket ") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 {
				desc := ""
				if len(parts) > 1 {
					desc = strings.TrimSpace(parts[1])
				}
				tickets = append(tickets, Ticket{
					Number:      len(tickets) + 1,
					Description: desc,
				})
			}
		}
	}

	return tickets, nil
}

func (m Model) runNextAgent() tea.Cmd {
	return func() tea.Msg {
		standardPrompt, err := os.ReadFile(m.StandardPromptPath)
		if err != nil {
			return tickMsg{output: "", err: err}
		}

		// Add kill file instruction to the prompt
		prompt := fmt.Sprintf("%s Please use the documentation in the input folder, especially the specification.md and the tickets.md. Please work on ticket %d. As your final task, create a file named 'killmenow.md' containing either 'success' or 'failure' to indicate whether you successfully completed the task.",
			string(standardPrompt), m.CurrentTicket+1)

		cmdParts := strings.Fields(m.CustomAgentCommand)
		if len(cmdParts) == 0 {
			return tickMsg{output: "", err: fmt.Errorf("invalid command")}
		}

		// Create log file for this agent
		now := time.Now()
		logFileName := fmt.Sprintf("%s-%s-party-agent-%d.log",
			strings.ToLower(now.Format("Monday")),
			now.Format("15-04-05"),
			m.CurrentTicket+1)
		
		logFile, err := os.Create(logFileName)
		if err != nil {
			return tickMsg{output: "", err: fmt.Errorf("failed to create log file: %w", err)}
		}

		// Append prompt as a command-line argument
		args := append(cmdParts[1:], prompt)
		cmd := exec.Command(cmdParts[0], args...)

		// Write initial info to log
		fmt.Fprintf(logFile, "=== Agent %d starting at %s ===\n", m.CurrentTicket+1, now.Format("15:04:05"))
		fmt.Fprintf(logFile, "Command: %s %s\n", cmdParts[0], strings.Join(args, " "))
		fmt.Fprintf(logFile, "Working directory: %s\n", func() string {
			if wd, err := os.Getwd(); err == nil {
				return wd
			}
			return "unknown"
		}())
		fmt.Fprintf(logFile, "Prompt: %s\n", prompt)
		fmt.Fprintf(logFile, "\n--- Agent Execution Started ---\n")
		fmt.Fprintf(logFile, "Note: Agent output is not captured here to allow file writes.\n")
		fmt.Fprintf(logFile, "Check party.sh and killmenow.md for agent results.\n")
		logFile.Close()

		// Start the command asynchronously WITHOUT redirecting output
		// This allows the agent to write to files normally
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

// View renders the current state of the model as a string for display
func (m Model) View() string {
	s := titleStyle.Render("Project Manager") + "\n\n"

	switch m.State {
	case StateFileCheck:
		s += m.renderFileCheck()

	case StateFileCheckResults:
		s += m.renderFileCheckResults()

	case StateFilePicker:
		s += m.renderFilePicker()

	case StateAgentSelection:
		s += m.renderAgentSelection()

	case StateCustomCommandEntry:
		s += m.renderCustomCommandEntry()

	case StateConfirmation:
		s += m.renderConfirmation()

	case StateRunning:
		s += m.renderRunning()

	case StateCompleted:
		s += m.renderCompleted()
	}

	return s
}

func (m Model) renderFileCheck() string {
	return "Checking for required files...\n"
}

func (m Model) renderFileCheckResults() string {
	s := "Checking for required files...\n\n"
	s += successStyle.Render("✅ Successfully found specification.md") + "\n"
	s += successStyle.Render("✅ Successfully found tickets.md") + "\n"
	s += successStyle.Render("✅ Successfully found standard-prompt.md") + "\n\n"
	s += infoStyle.Render("All files found! Press any key to continue...")
	return s
}

func (m Model) renderFilePicker() string {
	s := fmt.Sprintf("Missing file: %s\n", errorStyle.Render(m.MissingFiles[m.CurrentMissingIndex]))
	s += "Please select the file location:\n\n"
	s += m.FilePicker.View()
	return s
}

func (m Model) renderAgentSelection() string {
	s := "Select coding agent:\n\n"

	choices := []string{
		"claude --dangerously-skip-permissions",
		"Other (enter custom command)",
	}

	for i, choice := range choices {
		if i == m.SelectedAgent {
			s += selectedStyle.Render("→ "+choice) + "\n"
		} else {
			s += "  " + choice + "\n"
		}
	}

	s += "\n" + infoStyle.Render("Press Enter to continue")
	return s
}

func (m Model) renderCustomCommandEntry() string {
	s := "Enter custom agent command:\n\n"
	s += m.TextInput.View() + "\n\n"
	s += infoStyle.Render("Press Enter when done")
	return s
}

func (m Model) renderConfirmation() string {
	s := "Ready to start execution:\n\n"
	s += fmt.Sprintf("📁 Specification: %s\n", m.SpecificationPath)
	s += fmt.Sprintf("📋 Tickets: %s (%d tickets)\n", m.TicketsPath, len(m.Tickets))
	s += fmt.Sprintf("📝 Prompt: %s\n", m.StandardPromptPath)
	s += fmt.Sprintf("🤖 Agent: %s\n", m.CustomAgentCommand)
	s += fmt.Sprintf("⏱️  Delay between agents: %d seconds\n", m.DelaySeconds)
	s += "\n" + successStyle.Render("Press Enter to start")
	return s
}

func (m Model) renderRunning() string {
	s := fmt.Sprintf("Executing agents... (Ticket %d/%d)\n\n", m.CurrentTicket+1, len(m.Tickets))

	// Show ticket status with emojis
	for i, ticket := range m.Tickets {
		status, timeInfo := m.getTicketStatus(i, ticket)
		s += fmt.Sprintf("%s Ticket %d: %s%s\n", status, ticket.Number, ticket.Description, timeInfo)
	}

	if m.ProcessError != nil {
		s += "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.ProcessError)) + "\n"
	}
	return s
}

func (m Model) getTicketStatus(index int, ticket Ticket) (status, timeInfo string) {
	if index < m.CurrentTicket {
		// Completed tickets - show duration
		if ticket.Failed {
			status = "❌"
		} else {
			status = "✅"
		}
		duration := ticket.EndTime.Sub(ticket.StartTime)
		timeInfo = fmt.Sprintf(" - %s", formatDuration(duration))
	} else if index == m.CurrentTicket {
		// Current ticket
		if m.ProcessRunning {
			status = "🔄"
			// Show live duration for running ticket
			if !ticket.StartTime.IsZero() {
				currentDuration := time.Since(ticket.StartTime)
				timeInfo = fmt.Sprintf(" - %s", formatDuration(currentDuration))
			}
		} else if m.ProcessError != nil {
			status = "❌"
			// For failed current ticket, show duration if we have end time
			if !ticket.EndTime.IsZero() {
				duration := ticket.EndTime.Sub(ticket.StartTime)
				timeInfo = fmt.Sprintf(" - %s", formatDuration(duration))
			}
		} else if m.IsWaiting {
			remainingTime := int(time.Until(m.WaitingUntil).Seconds())
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
	return status, timeInfo
}

func (m Model) renderCompleted() string {
	s := successStyle.Render("All agents completed!") + "\n\n"

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
	s += "\nSummary:\n"
	s += fmt.Sprintf("✅ Successful: %d\n", successful)
	s += fmt.Sprintf("❌ Failed: %d\n", failed)
	s += fmt.Sprintf("📊 Total: %d\n", len(m.Tickets))
	s += fmt.Sprintf("⏱️  Total time: %s\n", formatDuration(totalDuration))

	s += "\n" + infoStyle.Render("Press q to quit")
	return s
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
