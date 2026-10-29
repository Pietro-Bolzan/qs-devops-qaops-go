# Quality gates

Cada gate é um job do workflow [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).
Um gate reprovado deixa o pipeline vermelho e, com a proteção da branch `main`, impede o merge do PR.

## Visão geral do fluxo

```text
push / pull request na main
        │
        ▼
  Gate 1 · Build ─┬─► Gate 2 · Testes ──► Gate 4 · Cobertura
                  └─► Gate 3 · Análise estática
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

## Gate 3 · Análise estática

| Campo                 | Registro |
|-----------------------|----------|
| **Nome e critério**   | Análise estática. Aprova se o `golangci-lint` (v2.15.0, regras em [`.golangci.yml`](../.golangci.yml)) reporta **zero problemas**. A formatação (`gofmt`/`goimports`) também é verificada. |
| **Justificativa**     | Risco R6: defeitos que compilam e podem passar nos testes, mas causam falhas silenciosas ou inseguras. Cada linter cobre um risco: `errcheck` (erro ignorado), `bodyclose` (conexão HTTP vazando), `gosec` (padrões inseguros, como servidor sem timeout), `errorlint` (comparação de erro que quebra com *wrap*), `staticcheck`/`unused`/`ineffassign` (código morto e bugs conhecidos), `gocyclo` (função com mais de 15 caminhos, difícil de testar). O critério é zero porque o projeto é pequeno e começou limpo: tolerar alguns problemas faria a dívida crescer sem ninguém perceber. |
| **Momento**           | Em paralelo ao Gate 2, depois do Gate 1, em todo `push`/`pull_request`. |
| **Evidência**         | Log do job, anotações nas linhas do código no PR e o artefato `relatorio-lint` (`golangci-lint.xml` no formato JUnit). |
| **Consequência**      | **Bloqueia.** |
| **Limitação/exceção** | Só encontra padrões conhecidos; não entende a regra de negócio. O `misspell` foi testado e descartado porque só conhece inglês e acusava falsos positivos nos comentários em português. Uma exceção pontual pode ser feita com `//nolint:<linter> // motivo`, sempre com justificativa no código e revisada no PR. |

**Achado ao ativar o gate:** a primeira execução apontou 35 problemas do `bodyclose` nos testes de
API. O helper `call` fechava o corpo, mas devolvia o `*http.Response`, o que deixava a garantia
invisível para o leitor e para a ferramenta. O helper foi refatorado para devolver só status e
cabeçalhos.

## Gate 4 · Cobertura

| Campo                 | Registro |
|-----------------------|----------|
| **Nome e critério**   | Cobertura de comandos. Aprova se o domínio (`internal/task`) tem **≥ 90%** e o total do código de aplicação (`internal/...`) tem **≥ 80%**. O cálculo é feito por [`scripts/coverage-gate.sh`](../scripts/coverage-gate.sh) sobre o `coverage.out` do Gate 2. |
| **Justificativa**     | Risco R7: regra crítica sem teste deixa uma regressão passar despercebida pelo Gate 2. O domínio concentra as regras RN01 a RN07, por isso tem o limite mais alto. 90% e não 100% porque perseguir 100% leva a testes artificiais, escritos só para "pintar" linhas. O limite total de 80% cobre a camada HTTP, onde alguns ramos de erro de infraestrutura (por exemplo, falha ao escrever a resposta) são caros de simular. Quando o gate foi criado, a cobertura era de 100% no domínio e 99,6% no total, então os limites deixam folga para refatorações sem permitir que um módulo inteiro fique sem teste. |
| **Momento**           | Depois do Gate 2 (reaproveita o perfil gerado por ele), em todo `push`/`pull_request`. |
| **Evidência**         | Tabela com a cobertura no resumo da execução, log do job e o artefato `relatorio-cobertura` (`coverage.html`, mostrando linha a linha o que não foi executado). |
| **Consequência**      | **Bloqueia.** |
| **Limitação/exceção** | Cobertura mede o que foi **executado**, não o que foi **verificado**: um teste sem asserção aumenta a cobertura sem proteger nada. Por isso o gate complementa o Gate 2 e a revisão dos testes no PR, mas não os substitui. O pacote `cmd/server` (só monta dependências e sobe o servidor) fica fora do cálculo; ele é exercitado manualmente na demonstração. Mudar um limite exige alterar o workflow em um PR com justificativa. |

## Como reproduzir localmente

```bash
make ci
```
