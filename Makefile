BIN := bin/tarefas
GOTESTSUM := go run gotest.tools/gotestsum@v1.13.0
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@v1.9.0

.DEFAULT_GOAL := help
.PHONY: help deps build run test test-report cover vet lint vuln secrets smoke fmt clean ci

help: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

deps: ## Baixa e verifica as dependências do módulo
	go mod download
	go mod verify

build: ## Compila o servidor em bin/tarefas
	go build -o $(BIN) ./cmd/server

run: ## Executa a API localmente (porta 8080, ou ADDR=:9090)
	go run ./cmd/server

test: ## Executa todos os testes com detector de condição de corrida
	go test -race -count=1 ./...

test-report: ## Executa os testes como no CI, gerando relatórios em reports/
	@mkdir -p reports
	$(GOTESTSUM) --format testname --junitfile reports/junit.xml --jsonfile reports/test-output.json \
		-- -race -count=1 -covermode=atomic -coverprofile=reports/coverage.out ./...

cover: test-report ## Verifica os limites de cobertura e gera reports/coverage.html
	go tool cover -html=reports/coverage.out -o reports/coverage.html
	./scripts/coverage-gate.sh reports/coverage.out

vet: ## Executa o go vet
	go vet ./...

lint: ## Executa a análise estática (golangci-lint, regras em .golangci.yml)
	$(GOLANGCI_LINT) run ./...

vuln: ## Procura vulnerabilidades conhecidas (govulncheck)
	$(GOVULNCHECK) ./...

secrets: ## Procura segredos no histórico do Git (gitleaks: brew install gitleaks)
	@command -v gitleaks >/dev/null || { echo "gitleaks não encontrado. Instale com: brew install gitleaks"; exit 1; }
	gitleaks git --redact --no-banner .

smoke: build ## Sobe o binário e executa o teste de fumaça
	./scripts/smoke-test.sh $(BIN)

fmt: ## Formata o código
	gofmt -s -w .

clean: ## Remove binários e relatórios gerados
	rm -rf bin reports coverage.out coverage.html

ci: deps vet build cover lint vuln secrets smoke ## Reproduz localmente os gates do pipeline
