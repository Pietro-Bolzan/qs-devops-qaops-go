BIN := bin/tarefas

.DEFAULT_GOAL := help
.PHONY: help deps build run test vet fmt clean

help: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

deps: ## Baixa e verifica as dependências do módulo
	go mod download
	go mod verify

build: ## Compila o servidor em bin/tarefas
	go build -o $(BIN) ./cmd/server

run: ## Executa a API localmente (porta 8080, ou ADDR=:9090)
	go run ./cmd/server

test: ## Executa todos os testes com detector de condição de corrida
	go test -race -count=1 ./...

vet: ## Executa o go vet
	go vet ./...

fmt: ## Formata o código
	gofmt -s -w .

clean: ## Remove binários e relatórios gerados
	rm -rf bin reports coverage.out coverage.html
