package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
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

type processOutputMsg string

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
}

type proceedToAgentSelectionMsg struct{}

type AppState int

const (
	StateFileCheck AppState = iota
	StateFileCheckResults
	StateFilePicker
	StateAgentSelection
	StateConfirmation
	StateRunning
	StateCompleted
)

type Model struct {
	State           AppState
	Width           int
	Height          int
	
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
	ProcessOutput    []string  // Changed to slice for better handling
	OutputScrollY    int       // Scroll position for output window
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
}

func initialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter custom agent command..."
	ti.CharLimit = 200
	
	return Model{
		State:              StateFileCheck,
		SpecificationPath:  "specifications/specification.md",
		TicketsPath:        "specifications/tickets.md",
		StandardPromptPath: "specifications/standard-prompt.md",
		MissingFiles:       []string{},
		TextInput:          ti,
		SelectedAgent:      0,
		Tickets:            []Ticket{},
		ProcessOutput:      []string{},
		DelaySeconds:       2, // Default 2 second delay between agents
	}
}

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
			if m.State == StateRunning && m.OutputScrollY > 0 {
				m.OutputScrollY--
			} else if m.State == StateAgentSelection {
				m.SelectedAgent = (m.SelectedAgent - 1 + 2) % 2
			}
			
		case "down", "j":
			if m.State == StateRunning && m.OutputScrollY < len(m.ProcessOutput)-5 {
				m.OutputScrollY++
			} else if m.State == StateAgentSelection {
				m.SelectedAgent = (m.SelectedAgent + 1) % 2
			}
			
		case "pgup":
			if m.State == StateRunning {
				m.OutputScrollY -= 10
				if m.OutputScrollY < 0 {
					m.OutputScrollY = 0
				}
			}
			
		case "pgdown":
			if m.State == StateRunning {
				m.OutputScrollY += 10
				maxScroll := len(m.ProcessOutput) - 5
				if m.OutputScrollY > maxScroll {
					m.OutputScrollY = maxScroll
				}
				if m.OutputScrollY < 0 {
					m.OutputScrollY = 0
				}
			}
		
		case "enter":
			switch m.State {
			case StateAgentSelection:
				if m.SelectedAgent == 0 {
					m.CustomAgentCommand = "claude --dangerously-skip-permissions"
					m.State = StateConfirmation
				} else {
					m.TextInput.Focus()
				}
			
			case StateConfirmation:
				m.ConfirmReady = true
			}
		}
		
		// Handle text input in agent selection
		if m.State == StateAgentSelection && m.SelectedAgent == 1 {
			var cmd tea.Cmd
			m.TextInput, cmd = m.TextInput.Update(msg)
			
			if msg.String() == "enter" && m.TextInput.Value() != "" {
				m.CustomAgentCommand = m.TextInput.Value()
				m.State = StateConfirmation
			}
			
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
		// Always show what we got
		m.ProcessOutput = append(m.ProcessOutput, fmt.Sprintf("=== TICK MSG DEBUG ==="))
		m.ProcessOutput = append(m.ProcessOutput, fmt.Sprintf("Output length: %d", len(msg.output)))
		m.ProcessOutput = append(m.ProcessOutput, fmt.Sprintf("Error: %v", msg.err))
		
		if msg.output != "" {
			// Split output into lines and store (keep empty lines)
			lines := strings.Split(msg.output, "\n")
			m.ProcessOutput = append(m.ProcessOutput, "=== AGENT OUTPUT ===")
			m.ProcessOutput = append(m.ProcessOutput, lines...)
		}
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

	case processOutputMsg:
		// Add new output line
		m.ProcessOutput = append(m.ProcessOutput, string(msg))
		// Auto-scroll to bottom if near the end
		if m.OutputScrollY >= len(m.ProcessOutput)-10 {
			m.OutputScrollY = len(m.ProcessOutput) - 5
			if m.OutputScrollY < 0 {
				m.OutputScrollY = 0
			}
		}
		return m, nil

	case processStartedMsg:
		// Store the running command
		m.CurrentCmd = msg.cmd
		m.ProcessOutput = append(m.ProcessOutput, fmt.Sprintf("Process started with PID: %d", msg.cmd.Process.Pid))
		// Start monitoring for kill file
		return m, checkForKillFile()

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
		m.ProcessOutput = append(m.ProcessOutput, fmt.Sprintf("Kill file found with content: %s", msg.content))
		
		// Kill the process
		if m.CurrentCmd != nil && m.CurrentCmd.Process != nil {
			m.ProcessOutput = append(m.ProcessOutput, "Terminating agent process...")
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
		
		// Clear output and error state for next agent
		m.ProcessOutput = []string{}
		m.OutputScrollY = 0
		m.ProcessError = nil
		m.ProcessRunning = true
		return m, m.runNextAgent()
	
	case time.Time:
		// Update the view to refresh the countdown
		if m.IsWaiting && time.Now().Before(m.WaitingUntil) {
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
				m.ProcessOutput = append(m.ProcessOutput, fmt.Sprintf("Error reading tickets: %v", err))
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
	content, err := ioutil.ReadFile(path)
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
		// Debug output
		debugOutput := fmt.Sprintf("Starting agent for ticket %d\n", m.CurrentTicket+1)
		debugOutput += fmt.Sprintf("Command: %s\n", m.CustomAgentCommand)
		
		standardPrompt, err := ioutil.ReadFile(m.StandardPromptPath)
		if err != nil {
			return tickMsg{output: debugOutput + fmt.Sprintf("Error reading prompt: %v", err), err: err}
		}
		
		// Add kill file instruction to the prompt
		prompt := fmt.Sprintf("%s Please use the documentation in the specifications folder, especially the specification.md and the tickets.md. Please work on ticket %d. As your final task, create a file named 'killmenow.md' containing either 'success' or 'failure' to indicate whether you successfully completed the task.",
			string(standardPrompt), m.CurrentTicket+1)
		
		debugOutput += fmt.Sprintf("Prompt length: %d\n", len(prompt))
		
		cmdParts := strings.Fields(m.CustomAgentCommand)
		if len(cmdParts) == 0 {
			return tickMsg{output: debugOutput + "Error: invalid command", err: fmt.Errorf("invalid command")}
		}
		
		// Append prompt as a command-line argument
		args := append(cmdParts[1:], prompt)
		cmd := exec.Command(cmdParts[0], args...)
		
		// Set up pipes for output
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return tickMsg{output: debugOutput + fmt.Sprintf("Error creating stdout pipe: %v", err), err: err}
		}
		
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return tickMsg{output: debugOutput + fmt.Sprintf("Error creating stderr pipe: %v", err), err: err}
		}
		
		debugOutput += fmt.Sprintf("Executing: %s with %d args\n", cmdParts[0], len(args))
		debugOutput += "--- COMMAND OUTPUT ---\n"
		
		// Start the command asynchronously
		if err := cmd.Start(); err != nil {
			return tickMsg{output: debugOutput + fmt.Sprintf("Error starting command: %v", err), err: err}
		}
		
		// Start goroutines to read output
		go streamOutput(stdout, "STDOUT")
		go streamOutput(stderr, "STDERR")
		
		// Return a message indicating the process has started
		return processStartedMsg{cmd: cmd}
	}
}

func streamOutput(pipe io.ReadCloser, prefix string) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		// In a real implementation, we'd send these lines to the update loop
		// For now, we'll just print them
		line := scanner.Text()
		_ = line // Suppress unused variable warning
		// TODO: Send processOutputMsg through a channel
	}
}

func checkForKillFile() tea.Cmd {
	return func() tea.Msg {
		return checkKillFileMsg{}
	}
}

func (m Model) View() string {
	s := titleStyle.Render("Project Manager") + "\n\n"
	
	switch m.State {
	case StateFileCheck:
		s += "Checking for required files...\n"
		
	case StateFileCheckResults:
		s += "Checking for required files...\n\n"
		s += successStyle.Render("✅ Successfully found specification.md") + "\n"
		s += successStyle.Render("✅ Successfully found tickets.md") + "\n"
		s += successStyle.Render("✅ Successfully found generic prompt.md") + "\n\n"
		s += infoStyle.Render("All files found! Press any key to continue...")
		
	case StateFilePicker:
		s += fmt.Sprintf("Missing file: %s\n", errorStyle.Render(m.MissingFiles[m.CurrentMissingIndex]))
		s += "Please select the file location:\n\n"
		s += m.FilePicker.View()
		
	case StateAgentSelection:
		s += "Select coding agent:\n\n"
		
		choices := []string{
			"claude --dangerously-skip-permissions",
			"Other (custom command)",
		}
		
		for i, choice := range choices {
			if i == m.SelectedAgent {
				s += selectedStyle.Render("→ " + choice) + "\n"
			} else {
				s += "  " + choice + "\n"
			}
		}
		
		if m.SelectedAgent == 1 {
			s += "\n" + m.TextInput.View()
		}
		
		s += "\n" + infoStyle.Render("Press Enter to continue")
		
	case StateConfirmation:
		s += "Ready to start execution:\n\n"
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
			if i < m.CurrentTicket {
				if ticket.Failed {
					status = "❌"
				} else {
					status = "✅"
				}
			} else if i == m.CurrentTicket {
				if m.ProcessRunning {
					status = "🔄"
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
			
			s += fmt.Sprintf("%s Ticket %d: %s\n", status, ticket.Number, ticket.Description)
		}
		
		// Show output window with border
		s += "\n" + strings.Repeat("─", 60) + "\n"
		s += "Output (↑/↓ to scroll, PgUp/PgDn for fast scroll):\n"
		s += strings.Repeat("─", 60) + "\n"
		
		// Show output with scrolling
		visibleLines := 15
		if len(m.ProcessOutput) > 0 {
			start := m.OutputScrollY
			end := start + visibleLines
			if end > len(m.ProcessOutput) {
				end = len(m.ProcessOutput)
			}
			if start < 0 {
				start = 0
			}
			
			for i := start; i < end; i++ {
				s += m.ProcessOutput[i] + "\n"
			}
		}
		
		s += strings.Repeat("─", 60) + "\n"
		
		if m.ProcessError != nil {
			s += errorStyle.Render(fmt.Sprintf("Error: %v", m.ProcessError)) + "\n"
		}
		
		if m.CurrentCmd != nil && m.CurrentCmd.Process != nil {
			s += infoStyle.Render(fmt.Sprintf("Process PID: %d", m.CurrentCmd.Process.Pid)) + "\n"
		}
		
	case StateCompleted:
		s += successStyle.Render("All agents completed!") + "\n\n"
		
		// Show final status
		successful := 0
		failed := 0
		for _, ticket := range m.Tickets {
			if ticket.Failed {
				failed++
			} else {
				successful++
			}
		}
		
		s += fmt.Sprintf("✅ Successful: %d\n", successful)
		s += fmt.Sprintf("❌ Failed: %d\n", failed)
		s += fmt.Sprintf("📊 Total: %d\n", len(m.Tickets))
		
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