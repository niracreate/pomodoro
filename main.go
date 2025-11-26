package main

import (
	"flag"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gen2brain/beeep"
)

// --- PIXEL BLOCK FONT ---
var bigDigits = map[rune][]string{
	'0': {"██████", "█    █", "█    █", "█    █", "██████"},
	'1': {"  ██  ", "  ██  ", "  ██  ", "  ██  ", "  ██  "},
	'2': {"██████", "     █", "██████", "█     ", "██████"},
	'3': {"██████", "     █", "██████", "     █", "██████"},
	'4': {"█    █", "█    █", "██████", "     █", "     █"},
	'5': {"██████", "█     ", "██████", "     █", "██████"},
	'6': {"██████", "█     ", "██████", "█    █", "██████"},
	'7': {"██████", "     █", "     █", "     █", "     █"},
	'8': {"██████", "█    █", "██████", "█    █", "██████"},
	'9': {"██████", "█    █", "██████", "     █", "██████"},
	':': {"      ", "  ██  ", "      ", "  ██  ", "      "},
}

// --- Styles ---
var (
	colorBlue   = lipgloss.Color("33")
	colorYellow = lipgloss.Color("220")
	colorSubtle = lipgloss.Color("241")

	styleContainer = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center)
	styleInput     = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorSubtle).Padding(1, 3).Width(40)
	styleHelp      = lipgloss.NewStyle().Foreground(colorSubtle).MarginTop(3)
)

// --- Model State ---
type sessionState int

const (
	stateSetup sessionState = iota
	stateRunning
)

type timerType int

const (
	typeWork timerType = iota
	typeBreak
)

type model struct {
	width  int
	height int

	state     sessionState
	timerType timerType
	paused    bool

	inputs     []textinput.Model
	focusIndex int

	workDuration  time.Duration
	breakDuration time.Duration
	timeLeft      time.Duration

	sessionsTotal  int
	currentSession int

	// <--- CHANGED: Added timerID to track unique timer loops
	timerID int
}

// --- Initialization ---

func initialModel(workArg, breakArg, sessArg string) model {
	m := model{
		inputs:  make([]textinput.Model, 3),
		timerID: 0, // <--- CHANGED: Initialize ID
	}

	t0 := textinput.New()
	t0.Placeholder = "Work (e.g. 25, 30s)"
	t0.Focus()
	t0.Width = 30
	t1 := textinput.New()
	t1.Placeholder = "Break (e.g. 5m)"
	t1.Width = 30
	t2 := textinput.New()
	t2.Placeholder = "Sessions (e.g. 4)"
	t2.Width = 30

	m.inputs[0] = t0
	m.inputs[1] = t1
	m.inputs[2] = t2

	if workArg != "" {
		m.state = stateRunning
		m.timerType = typeWork
		m.paused = false
		m.currentSession = 1
		m.workDuration = parseDurationInput(workArg, 25)
		m.breakDuration = parseDurationInput(breakArg, 5)
		s, _ := strconv.Atoi(sessArg)
		if s == 0 {
			s = 4
		}
		m.sessionsTotal = s
		m.timeLeft = m.workDuration

		// <--- CHANGED: Increment ID when starting immediately
		m.timerID++
	} else {
		m.state = stateSetup
		m.timerType = typeWork
	}

	return m
}

func (m model) Init() tea.Cmd {
	// <--- CHANGED: If quick start, ensure we start the tick loop with the ID
	if m.state == stateRunning {
		return tea.Batch(textinput.Blink, doTick(m.timerID))
	}
	return textinput.Blink
}

// --- Update Loop ---

// <--- CHANGED: tickMsg is now a struct containing the ID
type tickMsg struct {
	id int
}

// <--- CHANGED: doTick now accepts an ID
func doTick(id int) tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg{id: id}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	// <--- CHANGED: Check ID matches. If not, this is an old "ghost" tick.
	case tickMsg:
		if msg.id != m.timerID {
			return m, nil
		}

		if m.state == stateRunning && !m.paused && m.timeLeft > 0 {
			m.timeLeft -= time.Second
			if m.timeLeft <= 0 {
				return m.handleTimerFinish()
			}
			return m, doTick(m.timerID) // <--- CHANGED: Pass ID
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		if m.state == stateSetup {
			switch msg.String() {
			case "tab", "shift+tab", "enter", "up", "down":
				s := msg.String()
				if s == "enter" && m.focusIndex == len(m.inputs)-1 {
					return m.startTimer()
				}
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}
				if m.focusIndex > len(m.inputs)-1 {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs) - 1
				}

				cmds := make([]tea.Cmd, len(m.inputs))
				for i := 0; i <= len(m.inputs)-1; i++ {
					if i == m.focusIndex {
						cmds[i] = m.inputs[i].Focus()
						m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(colorBlue)
					} else {
						m.inputs[i].Blur()
						m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(colorSubtle)
					}
				}
				return m, tea.Batch(cmds...)
			}
		}

		if m.state == stateRunning {
			switch msg.String() {
			case " ":
				m.paused = !m.paused
				if !m.paused {
					// <--- CHANGED: Pass current ID when unpausing
					return m, doTick(m.timerID)
				}
			case "s":
				return m.handleTimerFinish()
			case "up":
				m.timeLeft += time.Minute
			case "down":
				if m.timeLeft > time.Minute {
					m.timeLeft -= time.Minute
				}
			}
		}
	}

	if m.state == stateSetup {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

// --- Helpers ---

func parseDurationInput(s string, defaultMin int) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Duration(defaultMin) * time.Minute
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if val, err := strconv.Atoi(s); err == nil {
		return time.Duration(val) * time.Minute
	}
	return time.Duration(defaultMin) * time.Minute
}

func playWindowsSound() {
	if runtime.GOOS == "windows" {
		go func() {
			_ = exec.Command("powershell", "-c", "(New-Object Media.SoundPlayer 'C:\\Windows\\Media\\Windows Notify System Generic.wav').PlaySync()").Run()
		}()
	} else {
		go beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
	}
}

func (m model) startTimer() (model, tea.Cmd) {
	m.workDuration = parseDurationInput(m.inputs[0].Value(), 30)
	m.breakDuration = parseDurationInput(m.inputs[1].Value(), 5)
	s, _ := strconv.Atoi(m.inputs[2].Value())
	if s == 0 {
		s = 4
	}
	m.sessionsTotal = s
	m.currentSession = 1
	m.state = stateRunning
	m.timerType = typeWork
	m.timeLeft = m.workDuration
	m.paused = false

	// <--- CHANGED: New session, New ID
	m.timerID++

	return m, doTick(m.timerID)
}

func (m model) handleTimerFinish() (model, tea.Cmd) {
	playWindowsSound()

	// <--- CHANGED: Increment ID. This invalidates any old ticks still in the pipeline.
	m.timerID++

	msg := ""
	if m.timerType == typeWork {
		msg = "Work session finished! Time for a break."
		_ = beeep.Notify("Pomodoro", msg, "")
		m.timerType = typeBreak
		m.timeLeft = m.breakDuration
	} else {
		msg = "Break finished! Back to work."
		_ = beeep.Notify("Pomodoro", msg, "")
		m.timerType = typeWork
		m.timeLeft = m.workDuration
		m.currentSession++
	}

	if m.currentSession > m.sessionsTotal {
		_ = beeep.Notify("Pomodoro", "All sessions completed!", "")
		return m, tea.Quit
	}

	// <--- CHANGED: Unpause automatically and start new tick loop with new ID
	m.paused = false
	return m, doTick(m.timerID)
}

// --- ASCII Renderer --- (No changes needed below)

func renderBigTime(d time.Duration, color lipgloss.Color) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d", minutes, seconds)
	height := 5
	lines := make([]string, height)
	for _, char := range timeStr {
		block, ok := bigDigits[char]
		if !ok {
			continue
		}
		for i := 0; i < height; i++ {
			lines[i] += block[i] + " "
		}
	}
	fullBlock := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Foreground(color).Render(fullBlock)
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	var s string
	if m.state == stateSetup {
		s = m.viewSetup()
	} else {
		s = m.viewTimer()
	}
	return styleContainer.Width(m.width).Height(m.height).Render(s)
}

func (m model) viewSetup() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorBlue).Render("POMODORO SETUP") + "\n\n")
	labels := []string{"Work Duration:", "Break Duration:", "Sessions:"}
	for i := range m.inputs {
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render(labels[i]) + "\n")
		b.WriteString(styleInput.Render(m.inputs[i].View()) + "\n\n")
	}
	b.WriteString(styleHelp.Render("\n[TAB] Switch  •  [ENTER] Start  •  [q] Quit"))
	return b.String()
}

func (m model) viewTimer() string {
	activeColor := colorBlue
	modeStr := fmt.Sprintf("WORK SESSION %d/%d", m.currentSession, m.sessionsTotal)
	if m.timerType == typeBreak {
		activeColor = colorYellow
		modeStr = "BREAK TIME"
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(activeColor).Render(modeStr)
	asciiTimer := lipgloss.NewStyle().Margin(1, 0).Render(renderBigTime(m.timeLeft, activeColor))
	status := "RUNNING"
	if m.paused {
		status = "PAUSED"
	}
	statusStr := lipgloss.NewStyle().Foreground(colorSubtle).Render(status)
	help := styleHelp.Render("\n[SPACE] Pause  •  [s] Skip  •  [↑/↓] +/- 1m  •  [q] Quit")
	return lipgloss.JoinVertical(lipgloss.Center, title, asciiTimer, statusStr, help)
}

func main() {
	flag.Parse()
	args := flag.Args()
	var w, b, s string
	if len(args) > 0 {
		w = args[0]
	}
	if len(args) > 1 {
		b = args[1]
	}
	if len(args) > 2 {
		s = args[2]
	}
	p := tea.NewProgram(initialModel(w, b, s), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}
