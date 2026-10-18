# API de Tarefas: pipeline DevOps/QAOps

API REST pequena, escrita em Go, para gerenciar tarefas (criar, listar, editar, mudar status e excluir).
O sistema é simples de propósito: o foco do trabalho é a **qualidade integrada ao fluxo de entrega**
(pipeline, testes automatizados e quality gates), não a complexidade da aplicação.

A caracterização do sistema e os riscos de qualidade estão em [docs/sistema.md](docs/sistema.md).

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

## Comandos

| Comando      | O que faz                                        |
|--------------|--------------------------------------------------|
| `make build` | Compila o binário em `bin/tarefas`               |
| `make run`   | Executa a API localmente                         |
| `make test`  | Executa os testes com detector de corrida        |
| `make vet`   | Executa o `go vet`                               |
| `make fmt`   | Formata o código                                 |
| `make clean` | Remove binários e relatórios                     |

## Estrutura

```text
cmd/server/      ponto de entrada da API
docs/            caracterização do sistema, riscos e relatórios
```

## Segurança

O repositório não contém senhas, tokens, chaves ou dados pessoais. Arquivos `.env` e chaves
estão no `.gitignore`. Se o pipeline precisar de algum segredo, ele deve ser cadastrado nos
*Secrets* da plataforma de CI, nunca no código.

## Integrantes

- Integrante 1: _nome_
- Integrante 2: _nome_
