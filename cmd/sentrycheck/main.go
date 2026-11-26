// Comando sentrycheck envia um evento de teste ao Sentry para conferir se a
// SENTRY_DSN está correta. Uso: SENTRY_DSN=... go run ./cmd/sentrycheck
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"tarefas/internal/monitor"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rep, err := monitor.New(monitor.Config{
		DSN:         os.Getenv("SENTRY_DSN"),
		Environment: envOr("SENTRY_ENVIRONMENT", "local"),
		Release:     "tarefas@sentrycheck",
		Logger:      logger,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	if !rep.Enabled() {
		fmt.Fprintln(os.Stderr, "SENTRY_DSN não definida: nada a enviar.")
		os.Exit(1)
	}

	rep.CaptureMessage("Evento de teste da API de Tarefas (cmd/sentrycheck)")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = rep.Close(ctx)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro: o evento não foi enviado a tempo:", err)
		os.Exit(1)
	}
	if rep.Failures() > 0 {
		fmt.Fprintln(os.Stderr, "erro: o Sentry não recebeu o evento (veja o aviso acima). Confira a SENTRY_DSN e a conexão.")
		os.Exit(1)
	}
	fmt.Println("Evento enviado. Confira em Issues no painel do Sentry.")
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
