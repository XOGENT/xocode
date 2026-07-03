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
	"github.com/xogent/xocode/internal/runner"
	"github.com/xogent/xocode/internal/stream"
	"github.com/xogent/xocode/internal/theme"
)

// role classifies a line in a conversation transcript.
type role int

const (
	roleUser role = iota
	roleAssistant
	roleTool
)

// chatMsg is one rendered turn in the planning/building transcript. rendered is
// a width-dependent cache cleared on resize.
type chatMsg struct {
	role     role
	text     string
	tool     string
	info     string
	rendered string
}

// Model is the root Bubble Tea model implementing the plan → review → build
// state machine as a conversation with claude followed by a cursor build.
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

	// composer is the persistent chat input (first message + follow-ups).
	composer textarea.Model

	// planning conversation
	sessionID    string
	taskTitle    string // first user message, used for the plan slug
	chat         []chatMsg
	planCaptured bool
	capturedPlan string

	// building
	buildLog     []chatMsg
	buildFiles   []string
	buildSummary string
	worktreeName string
	worktreePath string

	// streaming plumbing (shared by planning & building)
	spinner    spinner.Model
	chatVP     viewport.Model // transcript / build log
	reviewVP   viewport.Model // rendered plan markdown
	activeTool string
	running    bool
	turn       int
	turnStart  time.Time
	elapsed    time.Duration
	usage      stream.Usage
	ctx        context.Context
	cancel     context.CancelFunc
	events     <-chan stream.StreamEvent

	// plan
	store    *plan.Store
	plan     *plan.Plan
	planText string
	planPath string

	// history browser
	history    []plan.Plan
	histCursor int

	// settings editor
	setCursor int
	setDraft  config.Config

	// overlays
	showHelp bool

	err error

	// Injectable for tests; nil means use the real CLIs.
	startPlan  func(ctx context.Context, sessionID, message string, resume bool, turn int) tea.Cmd
	startBuild func(ctx context.Context, worktree, prompt string, turn int) tea.Cmd
}

// NewModel constructs the root model.
func NewModel() Model {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	th := theme.New()

	ta := textarea.New()
	ta.Placeholder = "Describe a change to plan and build — or just say hi."
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.Prompt = ""
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = th.Spinner

	h := help.New()
	h.ShowAll = true

	cfg := config.Load(wd)
	return Model{
		th:         th,
		cfg:        cfg,
		keys:       defaultKeys(),
		help:       h,
		state:      StatePreflight,
		checking:   true,
		projectDir: wd,
		composer:   ta,
		spinner:    sp,
		store:      plan.NewStore(cfg.PlanDir),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, runChecks(context.Background()), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
		// A help overlay swallows the next keypress to dismiss.
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		// Global shortcuts available when idle and inside the main flow.
		if !m.running && m.canNavigate() {
			switch {
			case matchesKey(msg, m.keys.Help):
				if !m.composerActive() {
					m.showHelp = true
					return m, nil
				}
			case matchesKey(msg, m.keys.History):
				return m.openHistory()
			case matchesKey(msg, m.keys.Settings):
				return m.openSettings()
			}
		}

	case preflightDoneMsg:
		m.checks = msg.results
		m.checking = false
		if doctor.AllOK(msg.results) {
			m.state = StateInput
			m.composer.Focus()
		}
		return m, nil

	case channelReadyMsg:
		if msg.turn != m.turn {
			return m, nil // stale launch from a cancelled turn
		}
		m.events = msg.ch
		m.running = true
		return m, tea.Batch(waitForEvent(m.events, msg.phase, msg.turn), m.spinner.Tick, tick())

	case streamEventMsg:
		if msg.turn != m.turn {
			return m, nil
		}
		m.applyEvent(msg.ev)
		return m, waitForEvent(m.events, m.state, msg.turn)

	case streamEOFMsg:
		return m.handleEOF(msg)

	case tickMsg:
		if m.running {
			m.elapsed = time.Since(m.turnStart)
			return m, tick()
		}
		return m, nil

	case editorFinishedMsg:
		if msg.err == nil && m.store != nil && m.plan != nil {
			if err := m.store.Reload(m.plan); err == nil {
				m.planText = m.plan.Text
				m.setReviewContent(m.planText)
			}
		}
		return m, nil

	case plansLoadedMsg:
		m.history = msg.plans
		m.histCursor = 0
		if msg.err != nil {
			m.err = msg.err
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

	switch m.state {
	case StatePreflight:
		return m.updatePreflight(msg)
	case StateInput, StatePlanning:
		return m.updateConversation(msg)
	case StateReview:
		return m.updateReview(msg)
	case StateBuilding:
		return m.updateBuilding(msg)
	case StateConfirmInit:
		return m.updateConfirmInit(msg)
	case StateSummary:
		return m.updateSummary(msg)
	case StateHistory:
		return m.updateHistory(msg)
	case StateSettings:
		return m.updateSettings(msg)
	case StateError:
		return m.updateError(msg)
	}
	return m, nil
}

// canNavigate reports whether global nav shortcuts should be honored.
func (m Model) canNavigate() bool {
	switch m.state {
	case StateInput, StatePlanning, StateReview, StateSummary:
		return true
	}
	return false
}

// composerActive reports whether the text composer currently owns keystrokes.
func (m Model) composerActive() bool {
	return (m.state == StateInput || m.state == StatePlanning) && !m.running
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

// updateConversation handles the landing composer and the planning chat.
func (m Model) updateConversation(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		// Mouse / other: let the transcript scroll.
		var cmd tea.Cmd
		m.chatVP, cmd = m.chatVP.Update(msg)
		return m, cmd
	}

	// While a turn streams, keys drive cancel / scroll only.
	if m.running {
		if key.String() == "esc" {
			return m.cancelTurn()
		}
		var cmd tea.Cmd
		m.chatVP, cmd = m.chatVP.Update(msg)
		return m, cmd
	}

	switch {
	case matchesKey(key, m.keys.Send):
		text := strings.TrimSpace(m.composer.Value())
		if text == "" {
			return m, nil
		}
		if m.state == StateInput {
			return m.beginPlanning(text)
		}
		return m.sendFollowUp(text)
	case matchesKey(key, m.keys.Newline):
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return m, cmd
	case key.String() == "esc" && m.state == StatePlanning:
		// Abandon the conversation and return to a fresh landing screen.
		return m.resetToInput(), nil
	}

	var cmd tea.Cmd
	m.composer, cmd = m.composer.Update(msg)
	return m, cmd
}

// beginPlanning opens a new planning session with the first message.
func (m Model) beginPlanning(message string) (tea.Model, tea.Cmd) {
	m.sessionID = runner.NewSessionID()
	m.taskTitle = message
	m.chat = []chatMsg{{role: roleUser, text: message}}
	m.state = StatePlanning
	m.composer.Reset()
	m.composer.Blur()
	return m.startTurn(message, false)
}

// sendFollowUp continues the existing session with another message.
func (m Model) sendFollowUp(message string) (tea.Model, tea.Cmd) {
	m.chat = append(m.chat, chatMsg{role: roleUser, text: message})
	m.composer.Reset()
	m.composer.Blur()
	return m.startTurn(message, true)
}

// startTurn launches one claude turn (new or resumed) and starts the stream.
func (m Model) startTurn(message string, resume bool) (tea.Model, tea.Cmd) {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.turn++
	m.running = true
	m.planCaptured = false
	m.capturedPlan = ""
	m.activeTool = ""
	m.usage = stream.Usage{}
	m.turnStart = time.Now()
	m.elapsed = 0
	m.err = nil
	m.layout()
	m.refreshChatViewport()

	var launch tea.Cmd
	if m.startPlan != nil {
		launch = m.startPlan(m.ctx, m.sessionID, message, resume, m.turn)
	} else {
		launch = startPlanning(m.ctx, m.projectDir, m.cfg.ClaudeModel, m.cfg.ClaudeEffort, m.sessionID, message, resume, m.turn)
	}
	return m, tea.Batch(launch, m.spinner.Tick, tick())
}

// cancelTurn stops the in-flight claude turn and returns to the idle composer.
func (m Model) cancelTurn() (tea.Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
	}
	m.running = false
	m.turn++ // invalidate any late events from the cancelled stream
	if len(m.chat) == 0 {
		return m.resetToInput(), nil
	}
	m.chat = append(m.chat, chatMsg{role: roleTool, tool: "cancelled", info: "turn stopped"})
	m.refreshChatViewport()
	m.composer.Focus()
	return m, textarea.Blink
}

// resetToInput returns to a clean landing screen, dropping the conversation.
func (m Model) resetToInput() Model {
	m.state = StateInput
	m.chat = nil
	m.sessionID = ""
	m.taskTitle = ""
	m.planCaptured = false
	m.capturedPlan = ""
	m.composer.Reset()
	m.composer.Focus()
	return m
}

// updateReview handles the plan review state.
func (m Model) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch {
		case matchesKey(key, m.keys.Approve):
			return m.beginBuilding()
		case matchesKey(key, m.keys.Edit):
			return m.editPlan()
		case matchesKey(key, m.keys.Refine):
			return m.refinePlan()
		case matchesKey(key, m.keys.Discard):
			return m.resetToInput(), nil
		case key.String() == "q":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.reviewVP, cmd = m.reviewVP.Update(msg)
	return m, cmd
}

// refinePlan returns to the conversation so the user can ask for plan changes;
// a new plan from claude replaces the current one.
func (m Model) refinePlan() (tea.Model, tea.Cmd) {
	m.state = StatePlanning
	m.composer.Placeholder = "What should change about the plan?"
	m.composer.Focus()
	m.refreshChatViewport()
	return m, textarea.Blink
}

// beginBuilding approves the plan and moves toward the build phase.
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
	m.buildLog = nil
	m.buildFiles = nil
	m.buildSummary = ""
	m.activeTool = ""
	m.usage = stream.Usage{}
	m.turn++
	m.running = true
	m.turnStart = time.Now()
	m.elapsed = 0
	m.err = nil
	m.layout()

	prompt := buildPrompt(m.planText)
	var launch tea.Cmd
	if m.startBuild != nil {
		launch = m.startBuild(m.ctx, m.worktreeName, prompt, m.turn)
	} else {
		launch = startBuilding(m.ctx, m.projectDir, m.cfg.CursorModel, m.worktreeName, prompt, m.turn)
	}
	return m, tea.Batch(launch, m.spinner.Tick, tick())
}

// updateBuilding handles keys while cursor streams.
func (m Model) updateBuilding(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		if m.cancel != nil {
			m.cancel()
		}
		m.running = false
		m.turn++
		m.state = StateReview
		return m, nil
	}
	var cmd tea.Cmd
	m.chatVP, cmd = m.chatVP.Update(msg)
	return m, cmd
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

// updateSummary handles the terminal summary screen.
func (m Model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q":
			return m, tea.Quit
		case "n", "enter":
			return m.resetToInput(), nil
		}
	}
	return m, nil
}

// openHistory loads saved plans and shows the browser.
func (m Model) openHistory() (tea.Model, tea.Cmd) {
	m.prev = m.state
	m.state = StateHistory
	return m, loadPlans(m.store)
}

// updateHistory handles the saved-plan browser.
func (m Model) updateHistory(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "esc", "q":
		m.state = StateInput
		m.composer.Focus()
		return m, nil
	case "up", "k":
		if m.histCursor > 0 {
			m.histCursor--
		}
	case "down", "j":
		if m.histCursor < len(m.history)-1 {
			m.histCursor++
		}
	case "enter":
		if m.histCursor < len(m.history) {
			p := m.history[m.histCursor]
			m.plan = &p
			m.planText = p.Text
			m.planPath = p.Path
			m.taskTitle = p.Task
			m.state = StateReview
			m.layout()
			m.setReviewContent(m.planText)
		}
	}
	return m, nil
}

// openSettings enters the settings editor with a draft copy of config.
func (m Model) openSettings() (tea.Model, tea.Cmd) {
	m.prev = m.state
	m.setDraft = m.cfg
	m.setCursor = 0
	m.state = StateSettings
	return m, nil
}

// updateError handles the terminal error screen.
func (m Model) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q":
			return m, tea.Quit
		case "esc", "enter":
			return m.resetToInput(), nil
		}
	}
	return m, nil
}

// handleEOF processes the closing of a subprocess event channel.
func (m Model) handleEOF(msg streamEOFMsg) (tea.Model, tea.Cmd) {
	if msg.turn != m.turn {
		return m, nil // stale EOF from a cancelled/superseded turn
	}
	m.running = false
	if m.err != nil {
		m.state = StateError
		return m, nil
	}
	switch msg.phase {
	case StatePlanning:
		if m.planCaptured && strings.TrimSpace(m.capturedPlan) != "" {
			return m.finalizePlan(m.capturedPlan)
		}
		// Conversational reply (e.g. "hi"): stay in the chat, never save.
		m.composer.Placeholder = "Reply, or describe the change you want planned."
		m.composer.Focus()
		m.refreshChatViewport()
		return m, textarea.Blink

	case StateBuilding:
		m.buildSummary = m.buildSummaryText()
		m.state = StateSummary
		return m, nil
	}
	return m, nil
}

// finalizePlan persists a captured plan and moves to review.
func (m Model) finalizePlan(text string) (tea.Model, tea.Cmd) {
	p := &plan.Plan{Task: m.taskTitle, Text: text, Model: m.cfg.ClaudeModel}
	if m.store != nil {
		if err := m.store.Save(p); err != nil {
			m.err = err
			m.state = StateError
			return m, nil
		}
	}
	m.plan = p
	m.planText = text
	m.planPath = p.Path
	m.state = StateReview
	m.layout()
	m.setReviewContent(text)
	return m, nil
}

// buildSummaryText prefers the result text, falling back to the last assistant
// line so the summary screen is never blank.
func (m Model) buildSummaryText() string {
	if s := strings.TrimSpace(m.buildSummary); s != "" {
		return s
	}
	for i := len(m.buildLog) - 1; i >= 0; i-- {
		if m.buildLog[i].role == roleAssistant {
			return m.buildLog[i].text
		}
	}
	return ""
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

// fileMutatingTools are the tool names whose targets we surface as changed files.
var fileMutatingTools = map[string]bool{
	"Edit": true, "Write": true, "MultiEdit": true, "NotebookEdit": true,
	"Create": true, "str_replace_editor": true, "apply_patch": true,
}

// applyEvent folds a streamed event into the transcript / plan and refreshes the
// active viewport.
func (m *Model) applyEvent(ev stream.StreamEvent) {
	building := m.state == StateBuilding
	switch ev.Kind {
	case stream.KindSystemInit:
		if m.sessionID == "" && ev.SessionID != "" {
			m.sessionID = ev.SessionID
		}
	case stream.KindAssistantText:
		msg := chatMsg{role: roleAssistant, text: stream.StripPlanMarkers(ev.Text)}
		if building {
			m.buildLog = append(m.buildLog, msg)
		} else if msg.text != "" {
			m.chat = append(m.chat, msg)
		}
	case stream.KindToolUse:
		m.activeTool = ev.ToolName
		entry := chatMsg{role: roleTool, tool: ev.ToolName, info: ev.ToolInfo}
		if building {
			m.buildLog = append(m.buildLog, entry)
			if fileMutatingTools[ev.ToolName] && ev.ToolInfo != "" {
				m.addBuildFile(ev.ToolInfo)
			}
		} else {
			m.chat = append(m.chat, entry)
		}
	case stream.KindPlan:
		m.planCaptured = true
		m.capturedPlan = ev.Text
	case stream.KindResult:
		m.usage = ev.Usage
		if building {
			m.buildSummary = ev.Text
		} else if !m.planCaptured {
			if p, ok := stream.ExtractPlan(ev.Text); ok {
				m.planCaptured = true
				m.capturedPlan = p
			}
		}
	case stream.KindError:
		m.err = errors.New(ev.Text)
	}
	if building {
		m.refreshBuildViewport()
	} else {
		m.refreshChatViewport()
	}
}

// addBuildFile records a changed file path, de-duplicated and order-preserving.
func (m *Model) addBuildFile(path string) {
	for _, f := range m.buildFiles {
		if f == path {
			return
		}
	}
	m.buildFiles = append(m.buildFiles, path)
}

// layout sizes the viewports and composer to the available content area.
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
		m.chatVP = viewport.New(contentW, contentH)
		m.reviewVP = viewport.New(contentW, contentH)
		m.ready = true
	}
	m.reviewVP.Width = contentW
	m.reviewVP.Height = max(1, contentH-1)

	// Composer occupies a few rows at the bottom of conversation screens.
	m.composer.SetWidth(contentW - 4)
	m.composer.SetHeight(3)
	// status bar (1) + composer box (3 rows + 2 border) = 6 reserved rows.
	m.chatVP.Width = contentW
	m.chatVP.Height = max(1, contentH-6)

	m.help.Width = contentW

	// Re-render width-dependent content.
	for i := range m.chat {
		m.chat[i].rendered = ""
	}
	for i := range m.buildLog {
		m.buildLog[i].rendered = ""
	}
	switch m.state {
	case StateInput, StatePlanning:
		m.refreshChatViewport()
	case StateBuilding:
		m.refreshBuildViewport()
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
