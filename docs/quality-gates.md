# Quality gates

Cada gate é um job do workflow [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).
Um gate reprovado deixa o pipeline vermelho e, com a proteção da branch `main`, impede o merge do PR.

## Visão geral do fluxo

```text
push / pull request na main
        │
        ▼
  Gate 1 · Build ──► Gate 2 · Testes
```

---

## Gate 1 · Build

| Campo                 | Registro |
|-----------------------|----------|
| **Nome e critério**   | Build. Aprova se `go mod verify`, `go mod tidy` (sem diferença no `go.mod`), `go build ./...` e `go vet ./...` terminam com código de saída 0. |
| **Justificativa**     | Risco R1: código que não compila ou dependências adulteradas/inconsistentes na `main` impedem qualquer pessoa de executar o sistema. O `go vet` detecta erros que o compilador aceita, como formatação de `Printf` com argumentos errados. |
| **Momento**           | Primeiro job do pipeline, em todo `push` na `main` e em todo `pull_request` para a `main`. Os outros gates dependem dele. |
| **Evidência**         | Log do job "Gate 1 · Build" e o artefato `tarefas-linux-amd64` com o binário gerado (guardado por 7 dias). |
| **Consequência**      | **Bloqueia.** Os jobs seguintes nem executam (`needs: build`). |
| **Limitação/exceção** | Só verifica que o código compila para Linux/amd64. Não garante que o comportamento esteja correto, isso é papel do Gate 2. Não há exceção: código que não compila nunca deve ser integrado. |

## Gate 2 · Testes

| Campo                 | Registro |
|-----------------------|----------|
| **Nome e critério**   | Testes. Aprova se **100% dos testes** (unitários e de API) passam, executados com `-race` (detector de condição de corrida) e `-count=1` (sem cache). |
| **Justificativa**     | Riscos R2 a R5: regressões nas regras de validação e no ciclo de status, quebra do contrato HTTP e condições de corrida. Todos os testes cobrem regras de negócio ou o contrato da API, então qualquer falha é uma regressão real. Por isso o critério é 100% e não um percentual menor. |
| **Momento**           | Depois do Gate 1, no mesmo evento (`push`/`pull_request`). |
| **Evidência**         | Log com o nome de cada teste (`gotestsum --format testname`), tabela no resumo da execução (total, falhas, tempo) e o artefato `relatorio-testes` com `junit.xml`, `test-output.json` e `coverage.out` (guardado por 30 dias, mesmo quando o gate falha). |
| **Consequência**      | **Bloqueia.** |
| **Limitação/exceção** | Os testes só verificam os cenários escritos. Um comportamento sem teste não é protegido (o Gate de cobertura, adicionado depois, mede essa lacuna). Não testam desempenho nem carga. Um teste instável (*flaky*) deve ser corrigido, nunca ignorado ou marcado com `t.Skip` para passar no pipeline. |

## Como reproduzir localmente

```bash
make ci
```
