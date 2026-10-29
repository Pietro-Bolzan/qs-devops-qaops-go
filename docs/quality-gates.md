# Quality gates

Cada gate é um job do workflow [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).
Um gate reprovado deixa o pipeline vermelho e, com a proteção da branch `main`, impede o merge do PR.

## Visão geral do fluxo

```text
push / pull request na main  (+ agenda semanal)
        │
        ▼
  Gate 1 · Build ─┬─► Gate 2 · Testes ──► Gate 4 · Cobertura ─┐
                  ├─► Gate 3 · Análise estática ──────────────┤
                  ├─► Gate 5 · Dependências ──────────────────┼─► Quality Gate (decisão)
                  └─► Gate 7 · Teste de fumaça ───────────────┤
  Gate 6 · Segredos (não depende do build) ───────────────────┘
```

Os gates 2, 3, 5 e 7 rodam em paralelo depois do build, e o Gate 6 começa junto com o build, para o
feedback chegar mais rápido. O job **Quality Gate**
roda sempre, mesmo quando algum gate falha. Ele monta no resumo da execução uma tabela com o
resultado de cada gate e só é aprovado se **todos** forem aprovados.

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
| **Limitação/exceção** | Cobertura mede o que foi **executado**, não o que foi **verificado**: um teste sem asserção aumenta a cobertura sem proteger nada. Por isso o gate complementa o Gate 2 e a revisão dos testes no PR, mas não os substitui. O pacote `cmd/server` (só monta dependências e sobe o servidor) fica fora do cálculo; ele é exercitado pelo Gate 7 (teste de fumaça do binário). Mudar um limite exige alterar o workflow em um PR com justificativa. |

## Gate 5 · Dependências

| Campo                 | Registro |
|-----------------------|----------|
| **Nome e critério**   | Vulnerabilidades em dependências. Aprova se `go mod verify` confirma a integridade dos módulos e o `govulncheck` (v1.9.0) **não encontra nenhuma vulnerabilidade alcançável** pelo código, seja na biblioteca padrão do Go ou em módulos de terceiros. |
| **Justificativa**     | Risco R8: usar um componente com vulnerabilidade conhecida (lista OWASP Top 10, item "Vulnerable and Outdated Components"). O `govulncheck` consulta a base oficial `vuln.go.dev` e analisa o grafo de chamadas: só reprova se o código **realmente chama** a função vulnerável, o que reduz falsos positivos em relação a ferramentas que olham só a versão. Como essa análise tem poucos falsos positivos, qualquer achado bloqueia. Hoje o projeto não usa módulos de terceiros, mas a biblioteca padrão também recebe correções de segurança, então o gate continua relevante (por isso o CI usa `check-latest: true`). |
| **Momento**           | Em paralelo aos Gates 2 e 3, depois do Gate 1, em todo `push`/`pull_request` **e toda segunda-feira às 06:00 UTC** (`schedule`), porque uma vulnerabilidade nova pode ser publicada sem nenhuma mudança no código. |
| **Evidência**         | Log do job e o artefato `relatorio-vulnerabilidades` (`govulncheck.txt`, com os pacotes e a versão do Go analisados). |
| **Consequência**      | **Bloqueia e exige revisão**: atualizar a dependência ou a versão do Go e registrar a análise no PR. |
| **Limitação/exceção** | Só conhece vulnerabilidades **já publicadas** na base do Go; falhas inéditas (*zero-day*) e problemas de lógica não são detectados. Não verifica licenças nem segredos no repositório (segredos são o papel do Gate 6). Se uma vulnerabilidade ainda não tiver correção, a exceção é documentar o risco no PR e ter aprovação dos dois integrantes. |

## Gate 6 · Segredos

| Campo                 | Registro |
|-----------------------|----------|
| **Nome e critério**   | Segredos versionados. Aprova se o `gitleaks` (v8.30.1, regras padrão da ferramenta) encontra **zero segredos** em **todos os commits** do histórico (senhas, tokens de API, chaves privadas, chaves de nuvem). |
| **Justificativa**     | Risco R9 e regra de segurança do enunciado: nunca versionar senhas, tokens ou chaves. Antes deste gate, o R9 dependia só do `.gitignore` e da atenção de quem revisava o PR, a única verificação manual entre os riscos. Um segredo removido em um commit posterior **continua vazado**, porque fica no histórico, por isso o gate analisa todos os commits (`fetch-depth: 0`) e não só os arquivos atuais. O critério é zero porque um único vazamento já obriga a revogar a credencial. |
| **Momento**           | Em todo `push`/`pull_request` e na agenda semanal. **Não depende do build**: começa junto com o Gate 1, já que não precisa compilar nada. |
| **Evidência**         | Log com cada achado (arquivo, linha, regra e commit, com o valor **mascarado** por `--redact`), tabela no resumo da execução (commits analisados e segredos encontrados) e o artefato `relatorio-segredos` (`gitleaks.json`). |
| **Consequência**      | **Bloqueia.** Além de corrigir o código, a credencial exposta precisa ser **revogada**, porque já pode ter sido copiada. |
| **Limitação/exceção** | Detecta padrões conhecidos e textos com alta entropia: pode deixar passar um segredo de formato incomum ou acusar um valor de teste como segredo. Um falso positivo é tratado registrando o *fingerprint* do achado em `.gitleaksignore`, com justificativa no PR. O binário é baixado com versão fixa e o SHA-256 é conferido antes da execução, para que a própria ferramenta não vire um risco de cadeia de suprimentos. |

## Gate 7 · Teste de fumaça

| Campo                 | Registro |
|-----------------------|----------|
| **Nome e critério**   | Teste de fumaça do binário. Aprova se o **binário gerado pelo Gate 1** sobe em até 10 s e as 10 verificações de [`scripts/smoke-test.sh`](../scripts/smoke-test.sh) passam: `GET /health` (200), criação válida (201) e inválida (422, RN01), consulta (200), ciclo `todo → in_progress → done` (200), reabrir tarefa concluída (409, RN07), tarefa inexistente (404) e desligamento gracioso com `SIGTERM` (código de saída 0). |
| **Justificativa**     | Risco R10: os testes do Gate 2 usam `httptest`, que chama o roteador em memória, sem rede e sem o `main`. Um erro na montagem do servidor (porta, variáveis de ambiente, timeouts, desligamento) passaria por todos os outros gates. Este gate testa **o mesmo artefato que seria entregue**, fazendo chamadas HTTP reais, como um cliente faria. As verificações são poucas de propósito: o teste de fumaça só confirma que o sistema "liga" e que o caminho principal funciona; os detalhes ficam com o Gate 2. |
| **Momento**           | Depois do Gate 1 (usa o artefato `tarefas-linux-amd64`), em paralelo aos Gates 2, 3 e 5, em todo `push`/`pull_request`. |
| **Evidência**         | Tabela "Gate 7 · Teste de fumaça" no resumo da execução (esperado × obtido de cada verificação), anotação `::error::` com a verificação que falhou e o artefato `relatorio-fumaca` (log JSON do servidor). |
| **Consequência**      | **Bloqueia.** |
| **Limitação/exceção** | Não mede desempenho nem carga e roda em um único sistema operacional (Linux). Os dados ficam em memória, então não verifica persistência. Uma verificação nova deve entrar só se proteger o caminho principal; o restante pertence aos testes de API do Gate 2. |

## Quality Gate (decisão final)

| Campo                 | Registro |
|-----------------------|----------|
| **Nome e critério**   | Aprova somente se os 7 gates terminaram com `success`. Um gate `failure`, `cancelled` ou `skipped` (que não rodou porque o anterior falhou) reprova. |
| **Justificativa**     | A proteção da branch exige o nome de cada check. Com um único check consolidado, adicionar um gate novo não exige mudar a configuração do GitHub, e o resultado fica resumido em uma tabela. |
| **Momento**           | Último job, sempre executado (`if: always()`). |
| **Evidência**         | Tabela "Decisão do Quality Gate" no resumo da execução. |
| **Consequência**      | **Bloqueia o merge** (check obrigatório). |
| **Limitação/exceção** | Depende da proteção da branch estar configurada (abaixo). Sem ela, o pipeline só avisa. |

## Proteção da branch main

O bloqueio do merge é configurado no GitHub (não fica em arquivo do repositório), por um
*ruleset* de branch:

1. **Settings → Rulesets → New ruleset → New branch ruleset**.
2. **Ruleset name:** `Proteção da main`. **Enforcement status:** `Active`. **Bypass list:** vazia
   (nem os administradores podem contornar).
3. **Target branches → Add target → Include default branch** (`main`).
4. Em **Rules**, ativar:
   - **Restrict deletions** e **Block force pushes**;
   - **Require a pull request before merging**, com *Allowed merge methods* = `Merge`;
   - **Require status checks to pass**, com o check **`Quality Gate`** (em *Add checks*) e a opção
     **Require branches to be up to date before merging**.
5. **Create**.

Com isso, nenhum código chega à `main` sem passar pelos 7 gates, nem mesmo o dos integrantes.

## Como reproduzir localmente

```bash
make ci
```
