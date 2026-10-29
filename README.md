# API de Tarefas: pipeline DevOps/QAOps

[![CI](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/actions/workflows/ci.yml)

API REST pequena, escrita em Go, para gerenciar tarefas (criar, listar, editar, mudar status e excluir).
O sistema é simples de propósito: o foco do trabalho é a **qualidade integrada ao fluxo de entrega**
(pipeline, testes automatizados e quality gates), não a complexidade da aplicação.

## Documentação

| Documento                                        | Conteúdo                                              |
|--------------------------------------------------|-------------------------------------------------------|
| [docs/sistema.md](docs/sistema.md)               | Finalidade, arquitetura, regras de negócio e riscos   |
| [docs/quality-gates.md](docs/quality-gates.md)   | Ficha de cada quality gate e proteção da branch       |
| [docs/experimentos.md](docs/experimentos.md)     | Cenários aprovado, reprovado e corrigido              |
| [docs/relatorio.md](docs/relatorio.md)           | Relatório técnico                                     |
| [docs/roteiro-demo.md](docs/roteiro-demo.md)     | Roteiro da demonstração                               |

## Requisitos

- [Go](https://go.dev/dl/) 1.27 ou superior
- `make` (opcional; todos os comandos têm equivalente com `go`)
- Nenhum banco de dados ou serviço externo: os dados ficam em memória

## Instalação e execução

```bash
git clone <url-do-repositorio>
cd <pasta-do-repositorio>
make deps    # ou: go mod download && go mod verify
make run     # ou: go run ./cmd/server
```

A API sobe em `http://localhost:8080`. Para usar outra porta: `ADDR=:9090 make run`.

Verificação rápida:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## API

Todas as requisições e respostas usam JSON. O prazo (`due_date`) usa o formato `AAAA-MM-DD`.

| Método   | Rota                  | Descrição                               | Sucesso | Erros            |
|----------|-----------------------|-----------------------------------------|---------|------------------|
| `GET`    | `/health`             | Verifica se a API está no ar            | 200     | -                |
| `GET`    | `/tasks?status=todo`  | Lista tarefas (filtro opcional)         | 200     | 422              |
| `POST`   | `/tasks`              | Cria tarefa                             | 201     | 400, 422         |
| `GET`    | `/tasks/{id}`         | Busca tarefa                            | 200     | 400, 404         |
| `PUT`    | `/tasks/{id}`         | Edita título, descrição, prioridade e prazo | 200 | 400, 404, 409, 422 |
| `PATCH`  | `/tasks/{id}/status`  | Muda o status                           | 200     | 400, 404, 409, 422 |
| `DELETE` | `/tasks/{id}`         | Remove tarefa                           | 204     | 400, 404         |

Significado dos erros: **400** JSON malformado ou id inválido; **404** tarefa inexistente;
**409** violação do ciclo de vida (transição proibida ou edição de tarefa concluída);
**422** regra de validação violada (o campo vem em `field`).

Exemplo:

```bash
curl -X POST localhost:8080/tasks \
  -d '{"title":"Configurar pipeline","priority":"high","due_date":"2026-12-01"}'

curl -X PATCH localhost:8080/tasks/1/status -d '{"status":"in_progress"}'
curl "localhost:8080/tasks?status=in_progress"
```

## Comandos

| Comando      | O que faz                                        |
|--------------|--------------------------------------------------|
| `make build` | Compila o binário em `bin/tarefas`               |
| `make run`   | Executa a API localmente                         |
| `make test`  | Executa testes unitários e de API (com `-race`)  |
| `make cover` | Verifica a cobertura e gera `reports/coverage.html` |
| `make vet`   | Executa o `go vet`                               |
| `make lint`  | Executa a análise estática (golangci-lint)       |
| `make vuln`  | Procura vulnerabilidades conhecidas (govulncheck) |
| `make fmt`   | Formata o código                                 |
| `make test-report` | Executa os testes como no CI e gera relatórios em `reports/` |
| `make ci`    | Reproduz localmente todos os gates do pipeline   |
| `make clean` | Remove binários e relatórios                     |

## Pipeline de CI

O workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) roda no GitHub Actions a cada
`push` na `main` e a cada pull request. Cada etapa funciona como um **quality gate**: se falhar,
o pipeline fica vermelho e o PR não pode ser integrado.

| Gate | Job                 | Critério de aprovação                            |
|------|---------------------|--------------------------------------------------|
| 1    | Gate 1 · Build      | Dependências verificadas, compila e `go vet` sem erros |
| 2    | Gate 2 · Testes     | 100% dos testes unitários e de API passando, com `-race` |
| 3    | Gate 3 · Análise estática | `golangci-lint` sem nenhum problema         |
| 4    | Gate 4 · Cobertura  | Domínio ≥ 90% e total (`internal/`) ≥ 80%        |
| 5    | Gate 5 · Dependências | `go mod verify` ok e `govulncheck` sem vulnerabilidades |
| -    | Quality Gate        | Decisão final: aprova só se os 5 gates foram aprovados |

O check **Quality Gate** é obrigatório na proteção da branch `main`, então um PR com qualquer gate
reprovado não pode ser integrado. O Gate 5 também roda toda segunda-feira, para pegar
vulnerabilidades publicadas depois do último commit.

As fichas completas de cada gate estão em [docs/quality-gates.md](docs/quality-gates.md).

## Estrutura

```text
cmd/server/        ponto de entrada: monta dependências e sobe o servidor
internal/task/     domínio: entidade, validações, ciclo de status, serviço e repositório
internal/httpapi/  camada HTTP: rotas, JSON e mapeamento de erros
scripts/           scripts usados pelo pipeline (gate de cobertura)
docs/              caracterização do sistema, quality gates e relatórios
.github/workflows/ pipeline de CI
```

## Segurança

O repositório não contém senhas, tokens, chaves ou dados pessoais. Arquivos `.env` e chaves
estão no `.gitignore`. Se o pipeline precisar de algum segredo, ele deve ser cadastrado nos
*Secrets* da plataforma de CI, nunca no código.

## Integrantes

| Integrante        | Responsável por                                                       |
|-------------------|-----------------------------------------------------------------------|
| Pietro Bolzan        | Base do repositório, domínio, serviço, API, testes, pipeline (Gates 1 e 2) |
| Eduardo dos Santos Lourenço | Análise estática, cobertura, dependências, Quality Gate, experimentos, relatório |
