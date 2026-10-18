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
	"tarefas/internal/task"
)

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

	svc := task.NewService(task.NewMemoryStore(), nil)
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(svc, logger),
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
