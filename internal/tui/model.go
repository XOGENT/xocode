package tui

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/xogent/xocode/internal/config"
	"github.com/xogent/xocode/internal/doctor"
	"github.com/xogent/xocode/internal/git"
	"github.com/xogent/xocode/internal/plan"
	"github.com/xogent/xocode/internal/stream"
	"github.com/xogent/xocode/internal/theme"
)

// logEntry is a structured record of one streamed event, re-rendered on resize.
type logEntry struct {
	kind stream.EventKind
	text string
	tool string
	info string
}

// Model is the root Bubble Tea model implementing the plan → review → build
// state machine.
type Model struct {
	th   theme.Theme
	cfg  config.Config
	keys keyMap
	help help.Model

	width  int
	height int
	ready  bool

	state State
	prev  State

	projectDir string

	// preflight
	checks   []doctor.Result
	checking bool

	// input
	task textarea.Model

	// streaming (shared by planning & building)
	spinner    spinner.Model
	vp         viewport.Model
	log        []logEntry
	activeTool string
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
	events     <-chan stream.StreamEvent

	// plan
	store    *plan.Store
	plan     *plan.Plan
	planText string
	planPath string

	// result text captured from the last KindResult event of either phase.
	resultText string

	// build
	buildSummary string
	worktreeName string
	worktreePath string

	err error

	// Injectable for tests; nil means use the real CLIs.
	startPlan  func(ctx context.Context, task string) tea.Cmd
	startBuild func(ctx context.Context, worktree, prompt string) tea.Cmd
}

// NewModel constructs the root model.
func NewModel() Model {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	th := theme.New()

	ta := textarea.New()
	ta.Placeholder = "Describe the task you want to plan and build…"
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = th.Spinner

	cfg := config.Load(wd)
	return Model{
		th:         th,
		cfg:        cfg,
		keys:       defaultKeys(),
		help:       help.New(),
		state:      StatePreflight,
		checking:   true,
		projectDir: wd,
		task:       ta,
		spinner:    sp,
		store:      plan.NewStore(cfg.PlanDir),
	}
}

func (m Model) Init() tea.Cmd {
	if m.state == StatePreflight {
		return tea.Batch(textarea.Blink, runChecks(context.Background()), m.spinner.Tick)
	}
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		// ctrl+c is a global quit; cancel any running subprocess first.
		if msg.String() == "ctrl+c" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}

	case preflightDoneMsg:
		m.checks = msg.results
		m.checking = false
		if doctor.AllOK(msg.results) {
			m.state = StateInput
		}
		return m, nil

	case channelReadyMsg:
		m.events = msg.ch
		m.running = true
		return m, tea.Batch(waitForEvent(m.events, msg.phase), m.spinner.Tick)

	case streamEventMsg:
		m.applyEvent(msg.ev)
		return m, waitForEvent(m.events, m.state)

	case streamEOFMsg:
		return m.handleEOF(msg)

	case editorFinishedMsg:
		if msg.err == nil && m.store != nil && m.plan != nil {
			if err := m.store.Reload(m.plan); err == nil {
				m.planText = m.plan.Text
				m.setReviewContent(m.planText)
			}
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = StateError
		m.running = false
		return m, nil

	case spinner.TickMsg:
		if m.running || m.checking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Per-state input handling.
	switch m.state {
	case StatePreflight:
		return m.updatePreflight(msg)
	case StateInput:
		return m.updateInput(msg)
	case StatePlanning, StateBuilding:
		return m.updateStreaming(msg)
	case StateReview:
		return m.updateReview(msg)
	case StateConfirmInit:
		return m.updateConfirmInit(msg)
	case StateSummary:
		return m.updateSummary(msg)
	case StateError:
		return m.updateError(msg)
	}
	return m, nil
}

// updateSummary handles the terminal summary screen.
func (m Model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "enter":
			return m, tea.Quit
		case "n":
			// Start another task.
			m.state = StateInput
			m.task.Reset()
			m.task.Focus()
			return m, nil
		}
	}
	return m, nil
}

// updatePreflight handles the prerequisite-check gate.
func (m Model) updatePreflight(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok || m.checking {
		return m, nil
	}
	switch key.String() {
	case "i":
		if len(m.checks) == 0 {
			return m, nil
		}
		m.checking = true
		return m, tea.Batch(remediate(context.Background(), m.checks), m.spinner.Tick)
	case "r":
		m.checking = true
		return m, tea.Batch(runChecks(context.Background()), m.spinner.Tick)
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// updateInput handles the task-entry state.
func (m Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+d" {
		task := strings.TrimSpace(m.task.Value())
		if task == "" {
			return m, nil
		}
		return m.beginPlanning(task)
	}
	var cmd tea.Cmd
	m.task, cmd = m.task.Update(msg)
	return m, cmd
}

// beginPlanning transitions into the planning state and launches claude.
func (m Model) beginPlanning(task string) (tea.Model, tea.Cmd) {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.state = StatePlanning
	m.log = nil
	m.planText = ""
	m.err = nil
	m.layout()
	if m.startPlan != nil {
		return m, m.startPlan(m.ctx, task)
	}
	return m, startPlanning(m.ctx, m.projectDir, m.cfg.ClaudeModel, m.cfg.ClaudeEffort, task)
}

// updateStreaming handles keys while a subprocess streams (planning/building).
func (m Model) updateStreaming(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		if m.cancel != nil {
			m.cancel()
		}
		// Return to input; the EOF from the cancelled process is handled there.
		m.running = false
		m.state = StateInput
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// updateReview handles the plan review state. Approve/build is wired in a later
// step; for now the reviewer can scroll and quit.
func (m Model) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "e":
			return m.editPlan()
		case "q":
			return m, tea.Quit
		case "enter":
			// Approve → build is wired in the build-phase step.
			return m.beginBuilding()
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// beginBuilding approves the plan and moves toward the build phase, first
// ensuring there's a git repo to isolate the work in.
func (m Model) beginBuilding() (tea.Model, tea.Cmd) {
	if !git.IsRepo(m.projectDir) {
		m.state = StateConfirmInit
		return m, nil
	}
	return m.buildStart()
}

// buildStart launches the build agent in a fresh worktree.
func (m Model) buildStart() (tea.Model, tea.Cmd) {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	repo := git.RepoName(m.projectDir)
	slug := ""
	if m.plan != nil {
		slug = m.plan.Slug
	}
	m.worktreeName = git.WorktreeName(slug, time.Now().Format("150405"))
	m.worktreePath = git.WorktreePath(repo, m.worktreeName)

	m.state = StateBuilding
	m.log = nil
	m.resultText = ""
	m.err = nil
	m.layout()

	prompt := buildPrompt(m.planText)
	if m.startBuild != nil {
		return m, m.startBuild(m.ctx, m.worktreeName, prompt)
	}
	return m, startBuilding(m.ctx, m.projectDir, m.cfg.CursorModel, m.worktreeName, prompt)
}

// updateConfirmInit handles the git-init confirmation prompt.
func (m Model) updateConfirmInit(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "y", "enter":
		if err := git.Init(m.projectDir); err != nil {
			m.err = err
			m.state = StateError
			return m, nil
		}
		return m.buildStart()
	case "n", "esc":
		m.state = StateReview
		return m, nil
	}
	return m, nil
}

func (m Model) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q":
			return m, tea.Quit
		case "esc", "enter":
			m.state = StateInput
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

// handleEOF processes the closing of a subprocess event channel.
func (m Model) handleEOF(msg streamEOFMsg) (tea.Model, tea.Cmd) {
	m.running = false
	if m.err != nil {
		m.state = StateError
		return m, nil
	}
	switch msg.phase {
	case StatePlanning:
		if strings.TrimSpace(m.resultText) == "" {
			m.err = errors.New("planning finished without producing a plan")
			m.state = StateError
			return m, nil
		}
		m.planText = m.resultText
		// Persist the plan, then show it in the review viewport.
		p := &plan.Plan{Task: m.task.Value(), Text: m.planText, Model: m.cfg.ClaudeModel}
		if m.store != nil {
			if err := m.store.Save(p); err != nil {
				m.err = err
				m.state = StateError
				return m, nil
			}
		}
		m.plan = p
		m.planPath = p.Path
		m.state = StateReview
		m.setReviewContent(m.planText)
		return m, nil

	case StateBuilding:
		m.buildSummary = m.resultText
		m.state = StateSummary
		return m, nil
	}
	return m, nil
}

// editPlan suspends the TUI and opens the plan document in $EDITOR.
func (m Model) editPlan() (tea.Model, tea.Cmd) {
	if m.plan == nil || m.planPath == "" {
		return m, nil
	}
	c := exec.Command(m.cfg.Editor, m.planPath) //nolint:gosec // editor from user env
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// applyEvent folds a streamed event into the log / plan text and refreshes the
// viewport.
func (m *Model) applyEvent(ev stream.StreamEvent) {
	switch ev.Kind {
	case stream.KindAssistantText:
		m.log = append(m.log, logEntry{kind: ev.Kind, text: ev.Text})
	case stream.KindToolUse:
		m.activeTool = ev.ToolName
		m.log = append(m.log, logEntry{kind: ev.Kind, tool: ev.ToolName, info: ev.ToolInfo, text: ev.Text})
	case stream.KindResult:
		// The result payload is the authoritative final text for the phase.
		m.resultText = ev.Text
	case stream.KindError:
		m.err = errors.New(ev.Text)
	}
	m.refreshStreamViewport()
}

// layout sizes the viewport to the available content area.
func (m *Model) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	headerH := lipgloss.Height(m.renderHeader())
	footerH := lipgloss.Height(m.renderFooter())
	contentH := m.height - headerH - footerH
	if contentH < 1 {
		contentH = 1
	}
	contentW := m.width

	if !m.ready {
		m.vp = viewport.New(contentW, contentH)
		m.ready = true
	} else {
		m.vp.Width = contentW
		m.vp.Height = contentH
	}

	m.task.SetWidth(contentW - 4)
	taskH := contentH - 6
	if taskH > 8 {
		taskH = 8
	}
	m.task.SetHeight(max(3, taskH))
	m.help.Width = contentW

	// Re-render content that depends on width.
	switch m.state {
	case StatePlanning, StateBuilding:
		m.refreshStreamViewport()
	case StateReview:
		m.setReviewContent(m.planText)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
