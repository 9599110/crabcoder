package repl

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"crabcoder/pkg/core/app"
	"crabcoder/pkg/service/ai"
)

var (
	senderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true)
	aiStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	thinkingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Italic(true)
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	bashStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))
	toolStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	welcomeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Bold(true)
	pendingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
)

type uiState int

const (
	stateReady uiState = iota
	stateStreaming
)

type model struct {
	app    *app.App
	ctx    context.Context
	cancel context.CancelFunc

	viewport   viewport.Model
	textarea   textarea.Model
	history    []string
	historyIdx int
	running    bool
	ready      bool

	state           uiState
	messages        []ai.Message
	stream          strings.Builder
	streamReasoning strings.Builder // DeepSeek 思考模式的思维链内容
	showThinking    bool            // 是否正在显示思考内容

	terminalWidth  int
	terminalHeight int
}

type streamMsg struct {
	event *ai.StreamEvent
	ch    <-chan *ai.StreamEvent
}

type streamDoneMsg struct{}

func listenStream(ch <-chan *ai.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		return streamMsg{event: event, ch: ch}
	}
}

func New(term interface{}, a *app.App) *model {
	return &model{
		app:        a,
		history:    make([]string, 0, 1000),
		historyIdx: -1,
		running:    false,
	}
}

func (m *model) Init() tea.Cmd {
	m.textarea = textarea.New()
	m.textarea.Placeholder = "输入消息 (/help 查看帮助, Ctrl+N 换行, ↑↓ 历史)"
	m.textarea.ShowLineNumbers = false
	m.textarea.CharLimit = 0
	m.textarea.SetHeight(3)
	m.textarea.FocusedStyle.CursorLine = lipgloss.NewStyle()
	m.textarea.BlurredStyle.CursorLine = lipgloss.NewStyle()

	m.viewport = viewport.New(0, 0)
	m.viewport.Style = lipgloss.NewStyle()

	m.running = true
	m.messages = make([]ai.Message, 0)

	m.appendOutput(welcomeStyle.Render("  CrabCoder - AI 编程助手"))
	m.appendOutput(helpStyle.Render("  /help 帮助  /exit 退出  /clear 清屏  /tools 工具列表  !cmd 执行命令"))

	return textarea.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case streamMsg:
		if m.state == stateStreaming {
			switch {
			case msg.event.Error != nil:
				m.appendOutput(errStyle.Render("AI 错误: " + msg.event.Error.Error()))
				m.state = stateReady

			case msg.event.Done:
				m.finishStream()
				return m, nil

			case msg.event.Type == "thinking":
				m.streamReasoning.WriteString(msg.event.ReasoningContent)
				thinkingText := m.streamReasoning.String()
				if !m.showThinking {
					m.showThinking = true
					m.appendOutput(thinkingStyle.Render("  [思考] " + thinkingText))
				} else {
					m.updateLastLine(thinkingStyle.Render("  [思考] " + thinkingText))
				}
				return m, listenStream(msg.ch)

			default:
				// content 类型
				if m.showThinking {
					m.appendOutput("") // 结束思考行
					m.showThinking = false
				}
				m.stream.WriteString(msg.event.Content)
				m.updateLastLine(aiStyle.Render(m.stream.String()))
				return m, listenStream(msg.ch)
			}
		}
		return m, nil

	case streamDoneMsg:
		if m.state == stateStreaming {
			m.state = stateReady
		}
		return m, nil

	case tea.KeyMsg:
		if m.state == stateStreaming {
			// 在流式输出期间忽略大部分按键
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				m.running = false
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.running = false
			return m, tea.Quit

		case tea.KeyEnter:
			if msg.Alt {
				m.textarea, _ = m.textarea.Update(msg)
				return m, nil
			}
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}
			m.textarea.Reset()
			m.historyIdx = -1
			m.appendOutput(senderStyle.Render("> " + input))

			return m, m.processInput(input)

		case tea.KeyUp:
			if len(m.history) == 0 {
				return m, nil
			}
			if m.historyIdx == -1 {
				m.historyIdx = len(m.history) - 1
			} else if m.historyIdx > 0 {
				m.historyIdx--
			}
			m.textarea.Reset()
			m.textarea.SetValue(m.history[m.historyIdx])
			m.textarea.CursorEnd()
			return m, nil

		case tea.KeyDown:
			if m.historyIdx == -1 {
				return m, nil
			}
			if m.historyIdx < len(m.history)-1 {
				m.historyIdx++
				m.textarea.Reset()
				m.textarea.SetValue(m.history[m.historyIdx])
			} else {
				m.historyIdx = -1
				m.textarea.Reset()
			}
			m.textarea.CursorEnd()
			return m, nil

		default:
			m.historyIdx = -1
		}

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.textarea.SetWidth(msg.Width)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
			m.textarea.SetWidth(msg.Width)
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if !m.ready {
		return "初始化中..."
	}

	sep := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#44475A")).
		Render(strings.Repeat("─", m.terminalWidth))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		sep,
		m.textarea.View(),
		helpStyle.Render("Enter 发送  Ctrl+N 换行  ↑↓ 历史  /exit 退出"),
	)
}

func (m *model) processInput(input string) tea.Cmd {
	m.history = append(m.history, input)

	switch {
	case input == "/exit" || input == "/quit":
		m.running = false
		return tea.Quit

	case input == "/help":
		m.showHelp()
		return nil

	case input == "/clear":
		m.viewport.SetContent("")
		m.messages = nil
		return nil

	case input == "/history":
		for i, h := range m.history {
			m.appendOutput(fmt.Sprintf("%d: %s", i+1, h))
		}
		return nil

	case input == "/tools":
		tools := m.app.Registry().List()
		m.appendOutput(outputStyle().Render("可用工具:"))
		for _, tool := range tools {
			m.appendOutput(fmt.Sprintf("  %s - %s", tool.Name(), tool.Description()))
		}
		return nil

	case strings.HasPrefix(input, "!"):
		m.appendOutput(bashStyle.Render("[命令执行: " + input[1:] + "] - 开发中"))
		return nil

	default:
		return m.startAIChat(input)
	}
}

func (m *model) startAIChat(input string) tea.Cmd {
	m.messages = append(m.messages, ai.Message{
		Role:    "user",
		Content: input,
	})

	m.stream.Reset()
	m.streamReasoning.Reset()
	m.showThinking = false
	m.appendOutput(aiStyle.Render(""))
	m.state = stateStreaming

	client := m.app.AIClient()

	req := &ai.ChatRequest{
		Model:    "",
		Messages: m.messages,
		System:   "你是一个 AI 编程助手，运行在 CrabCoder CLI 中。用中文回复。",
		Options: &ai.ChatOptions{
			ThinkingEnabled: true,
			ThinkingEffort:  "high",
		},
	}

	ch, err := client.Stream(m.ctx, req)
	if err != nil {
		m.appendOutput(errStyle.Render("AI 请求失败: " + err.Error()))
		m.state = stateReady
		return nil
	}

	return listenStream(ch)
}

func outputStyle() lipgloss.Style { return aiStyle }

func (m *model) showHelp() {
	m.appendOutput(helpStyle.Render(`
命令:
  /help      - 显示帮助
  /clear     - 清屏
  /history   - 命令历史
  /tools     - 工具列表
  /exit      - 退出

快捷键:
  Enter      - 发送消息
  Ctrl+N     - 换行
  ↑/↓        - 浏览历史
  Ctrl+R     - 搜索历史

Bash:
  !<cmd>     - 执行 Shell 命令
`))
}

func (m *model) appendOutput(text string) {
	current := m.viewport.View()
	if current != "" {
		text = current + "\n" + text
	}
	m.viewport.SetContent(text)
	m.viewport.GotoBottom()
}

func (m *model) updateLastLine(text string) {
	current := m.viewport.View()
	idx := strings.LastIndex(current, "\n")
	if idx >= 0 {
		text = current[:idx+1] + text
	}
	m.viewport.SetContent(text)
	m.viewport.GotoBottom()
}

func (m *model) Run(ctx context.Context) error {
	m.ctx = ctx
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func (m *model) finishStream() {
	content := m.stream.String()
	reasoning := m.streamReasoning.String()
	if content != "" || reasoning != "" {
		m.messages = append(m.messages, ai.Message{
			Role:             "assistant",
			Content:          content,
			ReasoningContent: reasoning,
		})
	}
	m.appendOutput("")
	m.state = stateReady
}

func (m *model) Stop() {
	m.running = false
}
