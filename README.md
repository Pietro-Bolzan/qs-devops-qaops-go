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
| `make vet`   | Executa o `go vet`                               |
| `make fmt`   | Formata o código                                 |
| `make clean` | Remove binários e relatórios                     |

## Estrutura

```text
cmd/server/        ponto de entrada: monta dependências e sobe o servidor
internal/task/     domínio: entidade, validações, ciclo de status, serviço e repositório
internal/httpapi/  camada HTTP: rotas, JSON e mapeamento de erros
docs/              caracterização do sistema, riscos e relatórios
```

## Segurança

O repositório não contém senhas, tokens, chaves ou dados pessoais. Arquivos `.env` e chaves
estão no `.gitignore`. Se o pipeline precisar de algum segredo, ele deve ser cadastrado nos
*Secrets* da plataforma de CI, nunca no código.

## Integrantes

- Integrante 1: _nome_
- Integrante 2: _nome_
