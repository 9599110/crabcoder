// CrabCoder - AI 编程助手 CLI
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"crabcoder/cmd/repl"
	"crabcoder/pkg/core/app"
	"crabcoder/pkg/logging"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	cfg, err := app.LoadConfig("")
	if err != nil {
		logging.Error("加载配置失败: %v", err)
		os.Exit(1)
	}

	logging.Info("启动 %s v%s", cfg.App.Name, cfg.App.Version)

	application, err := app.NewBuilder().
		WithConfig(cfg).
		WithDefaultTools().
		WithDefaultAI().
		Build()
	if err != nil {
		logging.Error("构建应用失败: %v", err)
		os.Exit(1)
	}

	r := repl.New(nil, application)

	if err := r.Run(ctx); err != nil {
		logging.Error("应用错误: %v", err)
		os.Exit(1)
	}
}
