package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dywoq/github_zip/pkg/config"
	"github.com/dywoq/github_zip/pkg/github"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

type tuiModel struct {
	spinner   spinner.Model
	config    *config.Config
	client    *github.Client
	repos     []string
	repoIndex int
	done      bool
	err       error
	logs      []string
	width     int
	height    int
}

func (m *tuiModel) Init() tea.Cmd {
	m.spinner.Spinner = spinner.Dot

	cfg, err := config.Load()
	if err != nil {
		m.err = err
		return tea.Quit
	}
	if cfg == nil {
		m.err = fmt.Errorf("Set GITHUB_TOKEN and GITHUB_USER environment variables")
		return tea.Quit
	}
	m.config = cfg
	m.client = github.NewClient(cfg.Token)

	if err := os.MkdirAll(cfg.BaseDir, 0755); err != nil {
		m.err = fmt.Errorf("Cannot create base dir: %v", err)
		return tea.Quit
	}

	repos, err := m.client.ListRepositories(context.Background(), m.config.User)
	if err != nil {
		m.err = err
		return tea.Quit
	}
	m.repos = repos

	return func() tea.Msg { return "process" }
}

func (m *tuiModel) processRepo() {
	if m.repoIndex >= len(m.repos) {
		m.done = true
		return
	}

	repo := m.repos[m.repoIndex]
	m.logs = append(m.logs, fmt.Sprintf("Processing: %s (%d/%d)", repo, m.repoIndex+1, len(m.repos)))

	err := m.client.ProcessRepository(context.Background(), m.config.User, repo, m.config.BaseDir)
	if err != nil {
		m.logs = append(m.logs, fmt.Sprintf("Error: %v", err))
	} else {
		m.logs = append(m.logs, fmt.Sprintf("Completed: %s", repo))
	}

	m.repoIndex++
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case string:
		if msg == "process" {
			m.processRepo()
			if !m.done {
				return m, func() tea.Msg { return "process" }
			}
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *tuiModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	if m.done {
		return successStyle.Render("All repositories processed!\n")
	}

	var s string
	s += infoStyle.Render(fmt.Sprintf("Processing %d/%d repositories\n", m.repoIndex, len(m.repos)))
	s += m.spinner.View() + "\n"

	if len(m.logs) > 0 {
		logHeight := m.height - 10
		if logHeight < 1 {
			logHeight = 1
		}
		start := len(m.logs) - logHeight
		if start < 0 {
			start = 0
		}
		for i := start; i < len(m.logs); i++ {
			s += m.logs[i] + "\n"
		}
	}

	return s
}

func runTUI() {
	p := tea.NewProgram(&tuiModel{})
	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCLI() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Println("Set GITHUB_TOKEN and GITHUB_USER environment variables")
		os.Exit(1)
	}

	if err := os.MkdirAll(cfg.BaseDir, 0755); err != nil {
		fmt.Printf("Cannot create base dir: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	client := github.NewClient(cfg.Token)

	repos, err := client.ListRepositories(ctx, cfg.User)
	if err != nil {
		fmt.Printf("Failed to list repositories: %v\n", err)
		os.Exit(1)
	}

	for _, repo := range repos {
		fmt.Println("Processing repo:", repo)
		if err := client.ProcessRepository(ctx, cfg.User, repo, cfg.BaseDir); err != nil {
			fmt.Printf("Failed to process repo %s: %v\n", repo, err)
		}
	}
}

func main() {
	tui := flag.Bool("tui", false, "Run with TUI interface")
	flag.Parse()

	if *tui {
		runTUI()
	} else {
		runCLI()
	}
}
