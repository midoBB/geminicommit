package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"

	"github.com/tfkhdyt/geminicommit/internal/service"
)

type action string

const (
	confirm    action = "CONFIRM"
	regenerate action = "REGENERATE"
	clue       action = "CLUE"
	edit       action = "EDIT"
	cancel     action = "CANCEL"
)

type appState int

const (
	stateViewing appState = iota
	stateSelecting
	stateInputting
)

type commitModel struct {
	viewport       viewport.Model
	state          appState
	ready          bool
	content        string
	cursor         int
	options        []option
	selectedAction action
	completed      bool
	inputText      string
	inputCursor    int
	editMode       bool
	width          int
	height         int
}

type option struct {
	text   string
	action action
}

// Message types for Bubble Tea
type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func newCommitModel(content string, editMode bool) *commitModel {
	options := []option{
		{"Yes", confirm},
		{"Regenerate", regenerate},
		{"Add Clue", clue},
		{"Edit", edit},
		{"Cancel", cancel},
	}

	if editMode {
		options = []option{
			{"Yes", confirm},
			{"Edit Again", edit},
			{"Regenerate", regenerate},
			{"Add Clue", clue},
			{"Cancel", cancel},
		}
	}

	return &commitModel{
		content:  content,
		state:    stateViewing,
		options:  options,
		cursor:   0,
		editMode: editMode,
	}
}

func (m *commitModel) Init() tea.Cmd {
	return nil
}

func (m *commitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg)
	case tickMsg:
		return m, tickCmd()
	}
	return m, nil
}

func (m *commitModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateViewing:
		return m.handleViewingKeys(msg)
	case stateSelecting:
		return m.handleSelectingKeys(msg)
	case stateInputting:
		return m.handleInputKeys(msg)
	}
	return m, nil
}

func (m *commitModel) handleViewingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.viewport.LineUp(1)
	case "down", "j":
		m.viewport.LineDown(1)
	case "tab", "enter":
		m.state = stateSelecting
	}
	return m, nil
}

func (m *commitModel) handleSelectingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}
	case "tab":
		m.state = stateViewing
	case "enter":
		selectedOption := m.options[m.cursor]
		if selectedOption.action == clue {
			m.state = stateInputting
			m.inputText = ""
			m.inputCursor = 0
		} else {
			m.selectedAction = selectedOption.action
			m.completed = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *commitModel) handleInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "escape":
		m.state = stateSelecting
	case "enter":
		m.selectedAction = clue
		m.completed = true
		return m, tea.Quit
	case "backspace":
		if m.inputCursor > 0 {
			m.inputText = m.inputText[:m.inputCursor-1] + m.inputText[m.inputCursor:]
			m.inputCursor--
		}
	case "left":
		if m.inputCursor > 0 {
			m.inputCursor--
		}
	case "right":
		if m.inputCursor < len(m.inputText) {
			m.inputCursor++
		}
	default:
		// Regular character input
		if len(msg.String()) == 1 {
			char := msg.String()
			m.inputText = m.inputText[:m.inputCursor] + char + m.inputText[m.inputCursor:]
			m.inputCursor++
		}
	}
	return m, nil
}

func (m *commitModel) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Measure actual component heights by rendering them - matches View() exactly
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F780E2")).
		Bold(true).
		Padding(0, 0, 1, 0)
	header := headerStyle.Render("Generated Commit Message:")

	navHintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		Italic(true).
		Padding(1, 0, 0, 0)

	var navHint string
	switch m.state {
	case stateViewing:
		navHint = navHintStyle.Render("↑/↓ to scroll • Tab/Enter to select options")
	case stateSelecting:
		navHint = navHintStyle.Render("↑/↓ to navigate • Enter to select • Tab to view message")
	case stateInputting:
		navHint = navHintStyle.Render("Type your clue • Enter to confirm • Escape to cancel")
	}

	// Render the content area to measure its height
	var contentArea string
	switch m.state {
	case stateViewing, stateSelecting:
		contentArea = m.renderMenuForMeasurement()
	case stateInputting:
		contentArea = m.renderInputForMeasurement()
	}

	// Calculate actual heights
	headerHeight := lipgloss.Height(header)
	navHintHeight := lipgloss.Height(navHint)
	contentHeight := lipgloss.Height(contentArea)
	borderPadding := 2 // Account for viewport border

	// Calculate available height for viewport
	usedHeight := headerHeight + navHintHeight + contentHeight + borderPadding
	viewportHeight := msg.Height - usedHeight

	// Ensure minimum height
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	if !m.ready {
		m.viewport = viewport.New(msg.Width-4, viewportHeight)
		m.viewport.SetContent(m.content)
		m.ready = true
	} else {
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = viewportHeight
	}

	return m, nil
}

// Helper methods to render components for measurement - must match actual rendering exactly
func (m *commitModel) renderMenuForMeasurement() string {
	// Simple menu container - matches renderMenu exactly
	menuStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Margin(1, 0)

	var menuItems strings.Builder
	for i, option := range m.options {
		cursor := " "
		if m.cursor == i && m.state == stateSelecting {
			cursor = ">"
		}

		itemStyle := lipgloss.NewStyle().Padding(0, 1)
		menuItems.WriteString(fmt.Sprintf("%s %s\n", cursor, itemStyle.Render(option.text)))
	}

	return menuStyle.Render(strings.TrimSuffix(menuItems.String(), "\n"))
}

func (m *commitModel) renderInputForMeasurement() string {
	// Simple input container - matches renderInput exactly
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F780E2")).
		Padding(1, 2).
		Margin(1, 0)

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F780E2")).
		Bold(true).
		Render("Enter your clue for the AI:")

	inputDisplay := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Margin(1, 0).
		Render("Sample text|") // Use sample text for measurement

	return inputStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, inputDisplay),
	)
}

func (m *commitModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Simple header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F780E2")).
		Bold(true).
		Padding(0, 0, 1, 0)

	header := headerStyle.Render("Generated Commit Message:")

	// Clean viewport styling
	viewportStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	if m.state == stateViewing {
		viewportStyle = viewportStyle.BorderForeground(lipgloss.Color("#F780E2"))
	} else {
		viewportStyle = viewportStyle.BorderForeground(lipgloss.Color("#555"))
	}

	viewportContent := viewportStyle.Render(m.viewport.View())

	// Simple navigation hints
	navHintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		Italic(true).
		Padding(1, 0, 0, 0)

	var navHint string
	switch m.state {
	case stateViewing:
		navHint = navHintStyle.Render("↑/↓ to scroll • Tab/Enter to select options")
	case stateSelecting:
		navHint = navHintStyle.Render("↑/↓ to navigate • Enter to select • Tab to view message")
	case stateInputting:
		navHint = navHintStyle.Render("Type your clue • Enter to confirm • Escape to cancel")
	}

	// Content area
	var contentArea string
	switch m.state {
	case stateViewing, stateSelecting:
		contentArea = m.renderMenu()
	case stateInputting:
		contentArea = m.renderInput()
	}

	// Simple vertical layout - no wrapper containers
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		viewportContent,
		navHint,
		contentArea,
	)
}

func (m *commitModel) renderMenu() string {
	// Simple menu container
	menuStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Margin(1, 0)

	if m.state == stateSelecting {
		menuStyle = menuStyle.BorderForeground(lipgloss.Color("#F780E2"))
	} else {
		menuStyle = menuStyle.BorderForeground(lipgloss.Color("#555"))
	}

	var menuItems strings.Builder
	for i, option := range m.options {
		cursor := " "
		if m.cursor == i && m.state == stateSelecting {
			cursor = ">"
		}

		// Simple item styling
		itemStyle := lipgloss.NewStyle().Padding(0, 1)
		if m.cursor == i && m.state == stateSelecting {
			itemStyle = itemStyle.Foreground(lipgloss.Color("#F780E2")).Bold(true)
		}

		menuItems.WriteString(fmt.Sprintf("%s %s\n", cursor, itemStyle.Render(option.text)))
	}

	return menuStyle.Render(strings.TrimSuffix(menuItems.String(), "\n"))
}

func (m *commitModel) renderInput() string {
	// Simple input container
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F780E2")).
		Padding(1, 2).
		Margin(1, 0)

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F780E2")).
		Bold(true).
		Render("Enter your clue for the AI:")

	// Show input text with cursor
	inputText := m.inputText
	if m.inputCursor <= len(inputText) {
		// Simple cursor
		cursor := "|"
		inputText = inputText[:m.inputCursor] + cursor + inputText[m.inputCursor:]
	}

	inputDisplay := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Margin(1, 0).
		Render(inputText)

	return inputStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, inputDisplay),
	)
}

func displayCommitMessageWithOptions(content string) (action, string) {
	return displayCommitMessageWithCustomOptions(content, false)
}

func displayCommitMessageWithEditOptions(content string) (action, string) {
	return displayCommitMessageWithCustomOptions(content, true)
}

func displayCommitMessageWithCustomOptions(content string, editMode bool) (action, string) {
	model := newCommitModel(content, editMode)

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running interface: %v\n", err)
		return cancel, ""
	}

	if m, ok := finalModel.(*commitModel); ok && m.completed {
		clueText := ""
		if m.selectedAction == clue {
			clueText = m.inputText
		}
		return m.selectedAction, clueText
	}

	return cancel, ""
}

type RootUsecase struct {
	gitService    *service.GitService
	geminiService *service.GeminiService
}

func NewRootUsecase(
	gitService *service.GitService,
	geminiService *service.GeminiService,
) *RootUsecase {
	return &RootUsecase{gitService, geminiService}
}

func (r *RootUsecase) RootCommand(stageAll *bool, promptAddition *string) error {
	if err := r.gitService.VerifyGitInstallation(); err != nil {
		return err
	}

	if err := r.gitService.VerifyGitRepository(); err != nil {
		return err
	}

	hasHook, _ := r.gitService.HasPreCommitHook()
	if hasHook {
		hookPath, _ := r.gitService.PreCommitHookPath()
		if r.gitService.IsExecutable(hookPath) {
			color.New(color.FgGreen).Println("✔ Running pre-commit hook...")
			err := r.gitService.RunPreCommitHook(hookPath)
			if err != nil {
				color.New(color.FgRed).Printf("Pre-commit hook failed: %v\n", err)
				return err
			}
			color.New(color.FgGreen).Println("✔ Pre-commit hook ran successfully.")
		}
	}
	if *stageAll {
		if err := r.gitService.StageAll(); err != nil {
			return err
		}
	}

	filesChan := make(chan []string, 1)
	deletedFilesChan := make(chan []string, 1)
	diffChan := make(chan string, 1)

	color.New(color.FgYellow).Print("Detecting staged files...")
	go func() {
		files, deletedFiles, diff, err := r.gitService.DetectDiffChanges()
		if err != nil {
			filesChan <- []string{}
			deletedFilesChan <- []string{}
			diffChan <- ""
			return
		}

		filesChan <- files
		deletedFilesChan <- deletedFiles
		diffChan <- diff
	}()

	underline := color.New(color.Underline)
	files, deletedFiles, diff := <-filesChan, <-deletedFilesChan, <-diffChan

	color.New(color.FgGreen).Println(" ✓")

	totalFiles := len(files) + len(deletedFiles)
	if totalFiles == 0 {
		return fmt.Errorf(
			"no staged changes found. stage your changes manually, or automatically stage all changes with the `--all` flag",
		)
	}

	if totalFiles == 1 {
		underline.Printf("Detected %d staged file:\n", totalFiles)
	} else {
		underline.Printf("Detected %d staged files:\n", totalFiles)
	}

	idx := 1
	for _, file := range files {
		color.New(color.Bold).Printf("     %d. %s\n", idx, file)
		idx++
	}

	for _, file := range deletedFiles {
		color.New(color.Bold, color.FgRed).Printf("     %d. %s (deleted)\n", idx, file)
		idx++
	}

generate:
	for {
		messageChan := make(chan string, 1)

		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2"))
		fmt.Print(titleStyle.Render("The AI is analyzing your changes..."))

		go func() {
			message, err := r.geminiService.AnalyzeChanges(context.Background(), diff, deletedFiles, promptAddition)
			if err != nil {
				messageChan <- ""
				return
			}

			messageChan <- message
		}()

		message := <-messageChan

		color.New(color.FgGreen).Println(" ✓")
		fmt.Print("\n")
		underline.Println("Changes analyzed!")

		if strings.TrimSpace(message) == "" {
			return fmt.Errorf("no commit messages were generated. try again")
		}

		selectedAction, clueText := displayCommitMessageWithOptions(message)

		switch selectedAction {
		case confirm:
			if err := r.gitService.CommitChanges(message); err != nil {
				return err
			}
			color.New(color.FgGreen).Println("✔ Successfully committed!")
			break generate
		case regenerate:
			continue
		case clue:
			if strings.TrimSpace(clueText) != "" {
				promptAddition = &clueText
				fmt.Print("\n")
				color.New(color.Italic).Println("Regenerating with provided clue...")
				fmt.Print("\n")
			} else {
				promptAddition = nil
			}
			continue generate
		case edit:
			for {
				tmpDir := os.TempDir()
				tmpFile, _ := os.CreateTemp(tmpDir, "COMMIT_EDITMSG")
				_ = os.WriteFile(tmpFile.Name(), []byte(message), 0o644)

				editor := os.Getenv("EDITOR")
				cmd := exec.Command(editor, tmpFile.Name())
				cmd.Dir = tmpDir
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					return err
				}

				msg, _ := os.ReadFile(tmpFile.Name())
				message = string(msg)
				_ = os.Remove(tmpFile.Name())

				underline.Print("Commit message edited!")
				fmt.Print("\n")
				selectedAction, clueText := displayCommitMessageWithEditOptions(message)

				switch selectedAction {
				case confirm:
					if err := r.gitService.CommitChanges(message); err != nil {
						return err
					}
					color.New(color.FgGreen).Println("✔ Successfully committed!")
					return nil
				case edit:
					continue
				case regenerate:
					continue generate
				case clue:
					if strings.TrimSpace(clueText) != "" {
						promptAddition = &clueText
						fmt.Print("\n")
						color.New(color.Italic).Println("Regenerating with provided clue...")
						fmt.Print("\n")
					} else {
						promptAddition = nil
					}
					continue generate
				case cancel:
					color.New(color.FgRed).Println("Commit cancelled")
					return nil
				}
			}
		case cancel:
			color.New(color.FgRed).Println("Commit cancelled")
			break generate
		}
	}

	return nil
}
