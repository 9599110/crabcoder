package app

import (
	"os"
	"strings"

	"crabcoder/pkg/core/bus"
	"crabcoder/pkg/core/state"
	"crabcoder/pkg/service/ai"
	"crabcoder/pkg/service/permission"
	"crabcoder/pkg/service/tool"
	"crabcoder/pkg/terminal"
	"crabcoder/pkg/tools"
)

type App struct {
	config      *Config
	terminal    terminal.Terminal
	registry    tools.Registry
	aiClient    ai.Client
	coordinator tool.Coordinator
	permission  permission.Checker
	bus         *bus.MessageBus
	state       state.Store[*state.AppState]
}

type Option func(*App)

func NewBuilder() *Builder {
	return &Builder{config: DefaultConfig()}
}

type Builder struct {
	config      *Config
	terminal    terminal.Terminal
	aiClient    ai.Client
	registry    tools.Registry
	permission  permission.Checker
	middlewares []Middleware
}

func (b *Builder) WithConfig(cfg *Config) *Builder              { b.config = cfg; return b }
func (b *Builder) WithTerminal(t terminal.Terminal) *Builder    { b.terminal = t; return b }
func (b *Builder) WithAI(client ai.Client) *Builder             { b.aiClient = client; return b }
func (b *Builder) WithRegistry(r tools.Registry) *Builder       { b.registry = r; return b }
func (b *Builder) WithPermission(p permission.Checker) *Builder { b.permission = p; return b }

func (b *Builder) WithMiddleware(m ...Middleware) *Builder {
	b.middlewares = append(b.middlewares, m...)
	return b
}

func (b *Builder) WithDefaultTools() *Builder {
	if b.registry == nil {
		b.registry = tools.NewRegistry()
	}
	tools.RegisterBaseTools(b.registry)
	tools.RegisterBashTool(b.registry)
	tools.RegisterDeleteTool(b.registry)
	return b
}

func (b *Builder) WithDefaultAI() *Builder {
	if b.aiClient == nil {
		provider, model, apiKey, baseURL := detectProvider(b.config)
		client, err := ai.NewClient(provider, model, apiKey, baseURL, b.config.Model.MaxTokens)
		if err != nil {
			client, _ = ai.NewClient("anthropic", "claude-sonnet-4-6", b.config.Model.APIKey, "", 8192)
		}
		b.aiClient = client
	}
	return b
}

func detectProvider(cfg *Config) (provider, model, apiKey, baseURL string) {
	// 从环境变量检测 Provider（优先 settings.json 注入的 env）
	baseURL = os.Getenv("MODEL_BASE_URL")
	if modelEnv := os.Getenv("MODEL"); modelEnv != "" {
		model = modelEnv
	} else {
		model = cfg.Model.Model
	}
	if token := os.Getenv("AUTH_TOKEN"); token != "" {
		apiKey = token
	} else {
		apiKey = cfg.Model.APIKey
	}

	if baseURL != "" {
		if strings.Contains(baseURL, "deepseek") {
			provider = "deepseek"
		} else if strings.Contains(baseURL, "openai") {
			provider = "openai"
		} else if strings.Contains(baseURL, "gemini") {
			provider = "gemini"
		} else {
			provider = "anthropic"
		}
	} else {
		provider = cfg.Model.Provider
		if provider == "" {
			provider = "anthropic"
		}
	}

	return
}

func (b *Builder) Build() (*App, error) {
	if b.aiClient == nil {
		return nil, ErrAIClientRequired
	}
	if b.registry == nil {
		b.registry = tools.NewRegistry()
	}
	if b.terminal == nil {
		b.terminal = terminal.NewDefault()
	}

	messageBus := bus.New()
	appState := state.NewStore(&state.AppState{
		Messages: make([]*state.Message, 0),
		Tasks:    make(map[string]*state.Task),
	})

	if b.permission == nil {
		b.permission = permission.NewManager(permission.PermissionConfig{
			Mode:        b.config.Permission.Mode,
			AlwaysAllow: b.config.Permission.AlwaysAllow,
			AlwaysDeny:  b.config.Permission.AlwaysDeny,
		})
	}

	coordinator := tool.NewCoordinator(b.registry, b.permission)

	app := &App{
		config:      b.config,
		terminal:    b.terminal,
		registry:    b.registry,
		aiClient:    b.aiClient,
		coordinator: coordinator,
		permission:  b.permission,
		bus:         messageBus,
		state:       appState,
	}

	for _, m := range b.middlewares {
		app.use(m)
	}

	return app, nil
}

func (a *App) use(m Middleware) {
	a.bus.Subscribe(m.Topic(), m.Handle)
}

func (a *App) Config() *Config                     { return a.config }
func (a *App) Terminal() terminal.Terminal         { return a.terminal }
func (a *App) Registry() tools.Registry            { return a.registry }
func (a *App) AIClient() ai.Client                 { return a.aiClient }
func (a *App) Coordinator() tool.Coordinator       { return a.coordinator }
func (a *App) Bus() *bus.MessageBus                { return a.bus }
func (a *App) State() state.Store[*state.AppState] { return a.state }
