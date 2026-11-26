// Comando server inicia a API HTTP do gerenciador de tarefas.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tarefas/internal/httpapi"
	"tarefas/internal/monitor"
	"tarefas/internal/task"
)

// version é a versão do binário, enviada ao Sentry como "release".
// No build de entrega: go build -ldflags "-X main.version=v1.0.0".
var version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("servidor encerrado com erro", "erro", err)
		os.Exit(1)
	}
}

// run monta as dependências, sobe o servidor e o encerra de forma
// graciosa ao receber SIGINT ou SIGTERM.
func run(logger *slog.Logger) error {
	addr := envOr("ADDR", ":8080")

	// Monitoramento de erros (Sentry). Sem SENTRY_DSN fica desligado.
	// A DSN vem do ambiente ou dos Secrets do CI, nunca do código.
	rep, err := monitor.New(monitor.Config{
		DSN:         os.Getenv("SENTRY_DSN"),
		Environment: envOr("SENTRY_ENVIRONMENT", "local"),
		Release:     "tarefas@" + version,
		Logger:      logger,
	})
	if err != nil {
		return err
	}
	logger.Info("monitoramento de erros", "sentry", rep.Enabled())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rep.Close(ctx); err != nil {
			logger.Warn("eventos do Sentry não enviados a tempo", "erro", err)
		}
	}()

	svc := task.NewService(task.NewMemoryStore(), nil)
	srv := &http.Server{
		Addr:              addr,
		Handler:           rep.Middleware(httpapi.New(svc, logger, httpapi.WithReporter(rep))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("servidor ouvindo", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("desligando servidor")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// envOr devolve o valor da variável de ambiente ou o padrão informado.
func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
