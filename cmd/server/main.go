// Comando server inicia a API HTTP do gerenciador de tarefas.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := envOr("ADDR", ":8080")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			log.Printf("erro ao responder health: %v", err)
		}
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("servidor ouvindo em %s", addr)
	log.Fatal(srv.ListenAndServe())
}

// envOr devolve o valor da variável de ambiente ou o padrão informado.
func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
