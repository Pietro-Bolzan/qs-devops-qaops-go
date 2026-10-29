# Relatório técnico: Pipeline de qualidade, automação e evidências

> **Rascunho.** Os trechos marcados com `_(preencher)_` dependem das execuções reais no GitHub
> Actions ou de dados da dupla. Para entregar, exportar para PDF/Word (6 a 10 páginas, sem anexos).

## 1. Identificação

| Campo        | Registro                                              |
|--------------|-------------------------------------------------------|
| Integrantes  | _(preencher: Integrante 1)_ e _(preencher: Integrante 2)_ |
| Sistema      | API de Tarefas: API REST em Go para gerenciar tarefas |
| Repositório  | _(preencher: URL do GitHub)_                          |
| Plataforma   | GitHub Actions (`ubuntu-latest`)                      |
| Data         | _(preencher)_                                         |

## 2. Contexto

A **API de Tarefas** cadastra tarefas com título, descrição, prioridade e prazo, e controla o ciclo
de vida de cada uma (`todo` → `in_progress` → `done`). O sistema foi construído pequeno de propósito:
o objeto de estudo é o fluxo de entrega e a qualidade integrada a ele, não as regras de negócio.

**Tecnologias.** Go 1.27, só com a biblioteca padrão (`net/http`, `encoding/json`, `log/slog`). Go
foi escolhido porque já traz testes, cobertura, detector de corrida e `go vet` nativos, compila
rápido (o que encurta o ciclo de feedback do pipeline) e gera um binário único. Não usar bibliotecas
externas reduz a superfície de ataque e a manutenção de dependências. As consequências disso para o
Gate 5 estão discutidas na seção 8.

**Arquitetura.** Três camadas, cada uma dependendo só da de baixo:

```text
cmd/server        → monta as dependências, sobe o servidor HTTP e faz desligamento gracioso
internal/httpapi  → rotas, validação do JSON, tradução de erros de domínio em status HTTP
internal/task     → entidade Tarefa, regras de validação, ciclo de status,
                    serviço (casos de uso) e repositório em memória
```

O domínio não conhece HTTP e recebe o relógio por injeção (`Clock`). Por isso as regras podem ser
testadas de forma determinística, com uma data fixa, sem depender do dia em que o pipeline roda.

**Funcionalidade estudada.** O ciclo de vida da tarefa, regido pelas regras:

- RN01: título obrigatório, de 3 a 120 caracteres.
- RN02: descrição de até 1000 caracteres.
- RN03: prioridade `low`, `medium` ou `high` (padrão `medium`).
- RN04: prazo opcional e nunca anterior à data atual.
- RN05: toda tarefa nasce como `todo`.
- RN06: transições permitidas: `todo→in_progress`, `in_progress→todo` e `in_progress→done`.
- RN07: `done` é final: a tarefa não muda de status nem pode ser editada.

## 3. Riscos

Os riscos foram levantados **antes** de escolher as ferramentas. Cada verificação do pipeline
existe para controlar ao menos um deles.

| ID | Risco                                                      | Verificação                     | Gate |
|----|------------------------------------------------------------|---------------------------------|------|
| R1 | Código que não compila ou dependências inconsistentes      | `go build`, `go vet`, `go mod verify` | 1 |
| R2 | Validação quebrada aceita dados inválidos                  | Testes unitários                | 2    |
| R3 | Transição de status indevida (ex.: reabrir tarefa concluída) | Testes unitários e de API     | 2    |
| R4 | API responde com status ou JSON fora do contrato           | Testes de API/integração        | 2    |
| R5 | Condição de corrida com requisições simultâneas            | Testes concorrentes com `-race` | 2    |
| R6 | Erros ignorados, código morto, padrões inseguros           | `golangci-lint`                 | 3    |
| R7 | Regra crítica sem nenhum teste                             | Cobertura por módulo            | 4    |
| R8 | Componente com vulnerabilidade conhecida                   | `govulncheck`                   | 5    |
| R9 | Segredo versionado no repositório                          | `.gitignore` + revisão do PR    | manual |

R9 é o único risco sem verificação automática, o que está discutido como lacuna na seção 8.

## 4. Pipeline

**Gatilhos.** `push` na `main`, `pull_request` para a `main`, execução manual (`workflow_dispatch`)
e agenda semanal (segunda-feira às 06:00 UTC), porque vulnerabilidades novas são publicadas mesmo
sem mudança no código. Um push novo no mesmo PR cancela a execução anterior (`concurrency`), o que
economiza minutos de CI.

**Etapas e fluxo de decisão.**

```text
push / PR ─► Gate 1 · Build ─┬─► Gate 2 · Testes ──► Gate 4 · Cobertura ─┐
                             ├─► Gate 3 · Análise estática ──────────────┼─► Quality Gate
                             └─► Gate 5 · Dependências ──────────────────┘   (decisão final)
```

| Etapa (job)               | Ferramentas                         | Evidência gerada                          |
|---------------------------|-------------------------------------|-------------------------------------------|
| Gate 1 · Build            | `go mod download/verify/tidy`, `go build`, `go vet` | Log + binário `tarefas-linux-amd64` |
| Gate 2 · Testes           | `go test -race` via `gotestsum`     | `junit.xml`, `test-output.json`, `coverage.out`, tabela no resumo |
| Gate 3 · Análise estática | `golangci-lint` v2.15.0             | Anotações no PR + `golangci-lint.xml`     |
| Gate 4 · Cobertura        | `go tool cover` + `scripts/coverage-gate.sh` | `coverage.html` + tabela no resumo |
| Gate 5 · Dependências     | `go mod verify` + `govulncheck` v1.9.0 | `govulncheck.txt`                      |
| Quality Gate              | Script de consolidação              | Tabela "Decisão do Quality Gate"          |

**Decisões de projeto do pipeline:**

- **Gates em paralelo.** Os Gates 2, 3 e 5 só dependem do build, então rodam ao mesmo tempo e o
  feedback chega mais rápido. O Gate 4 depende do 2 porque reaproveita o perfil de cobertura dele,
  sem rodar os testes duas vezes.
- **Um único check obrigatório.** O job `Quality Gate` roda sempre (`if: always()`) e só aprova se
  todos os gates foram aprovados. A proteção da branch `main` exige apenas esse check, então
  adicionar um gate novo não exige mudar a configuração do GitHub.
- **Versões fixadas.** Go (via `go.mod`), `gotestsum`, `golangci-lint` e `govulncheck` têm versão
  fixa. Isso torna o pipeline reproduzível: um gate não muda de comportamento sozinho porque uma
  ferramenta foi atualizada.
- **Privilégio mínimo.** `permissions: contents: read`. O pipeline não usa nenhum segredo.
- **Reprodução local.** `make ci` executa os mesmos gates na máquina do desenvolvedor, para que a
  falha seja descoberta antes do push.

## 5. Testes

**Estratégia.** Os testes foram escolhidos a partir dos riscos R2 a R5, e não pela cobertura. Cada
regra de negócio tem casos **positivos** (o que deve ser aceito, incluindo os valores-limite) e
**negativos** (o que deve ser rejeitado). Foram usadas técnicas de análise de valor-limite (título
com 2, 3, 120 e 121 caracteres; prazo ontem, hoje e amanhã) e de tabela de decisão (as 9
combinações de transição de status).

| Nível            | Arquivo(s)                                   | Funções / casos | O que verifica |
|------------------|----------------------------------------------|-----------------|----------------|
| Unitário (domínio)  | `internal/task/task_test.go`              | 8 / 32*         | RN01 a RN07, valores-limite, todas as transições |
| Unitário (repositório) | `internal/task/store_test.go`          | 6               | IDs sequenciais, não encontrado, ordenação, isolamento de cópias, criação concorrente |
| Unitário (serviço) | `internal/task/service_test.go`            | 11              | Casos de uso, filtro, propagação de erro do repositório, transição concorrente |
| API / integração | `internal/httpapi/handler_test.go`           | 13 / 23         | Contrato HTTP de todos os endpoints (status, cabeçalhos, JSON), ciclo de vida completo, erro interno sem vazar detalhes |
| **Total**        |                                              | **38 / 72**     | |

\* Casos contando os subtestes de tabela. _(conferir com o resumo do Gate 2)_

**Destaques:**

- **Relógio injetado.** Os testes usam a data fixa 18/10/2026. Sem isso, o teste de "prazo não pode
  ser no passado" passaria ou falharia dependendo do dia, ou seja, seria um teste instável.
- **Testes de API com `httptest`.** Sobem a aplicação completa (handler + serviço + repositório) em
  memória, sem porta de rede fixa, o que os torna rápidos e confiáveis no CI.
- **Concorrência.** Dois testes disparam de 50 a 100 *goroutines* ao mesmo tempo: um confirma que não
  surgem IDs duplicados e o outro que só uma de várias transições simultâneas para `done` é aceita.
  Com `-race`, qualquer acesso não sincronizado reprova o gate.
- **Falha de infraestrutura simulada.** Um repositório falso que sempre falha confirma que o serviço
  propaga o erro e que a API responde `500` sem expor a mensagem interna.

**Categorias não aplicáveis.** Não há testes de interface (o sistema não tem interface gráfica) nem
de banco de dados (a persistência é em memória). Testes de carga e de desempenho ficaram fora do
escopo (seção 8).

## 6. Quality gates

As fichas completas estão em `docs/quality-gates.md`. Resumo:

| Gate | Critério | Risco | Consequência | Principal limitação |
|------|----------|-------|--------------|---------------------|
| 1 · Build | Dependências íntegras, `go.mod` organizado, compila e `go vet` sem erros | R1 | Bloqueia | Não diz nada sobre o comportamento |
| 2 · Testes | 100% dos testes passando, com `-race` | R2 a R5 | Bloqueia | Só protege o que foi testado |
| 3 · Análise estática | Zero problemas no `golangci-lint` | R6 | Bloqueia | Só pega padrões conhecidos |
| 4 · Cobertura | Domínio ≥ 90%, total ≥ 80% | R7 | Bloqueia | Mede execução, não verificação |
| 5 · Dependências | `govulncheck` sem vulnerabilidade alcançável | R8 | Bloqueia e exige revisão | Só vulnerabilidades já publicadas |

**Justificativa dos limites:**

- **100% dos testes no Gate 2.** Todo teste existente cobre uma regra de negócio ou o contrato da
  API. Tolerar uma falha seria aceitar uma regressão conhecida.
- **90% e 80% no Gate 4.** Quando o gate foi criado, a cobertura era 100% no domínio e 99,6% no
  total. Os limites deixam folga para refatorar sem exigir testes artificiais, mas impedem que um
  módulo novo entre sem teste. O domínio tem limite maior porque concentra as regras RN01 a RN07.
- **Zero problemas no Gate 3.** O projeto começou limpo. Aceitar "até N problemas" faria a dívida
  técnica crescer sem ninguém perceber.

**Achado ao ativar o Gate 3.** A primeira execução do `golangci-lint` apontou 35 ocorrências do
`bodyclose` nos testes de API e 1 do `misspell`. A primeira era um problema real de clareza: o
helper de teste fechava o corpo da resposta, mas devolvia o `*http.Response`, o que tornava a
garantia invisível. Ele foi refatorado. A segunda era um falso positivo: o `misspell` só conhece
inglês e acusou "Comando" nos comentários em português. Esse linter foi removido, com a
justificativa registrada na configuração. Isso mostra que a configuração de um gate também precisa
de avaliação crítica, não só de cópia.

## 7. Experimentos

Detalhes e evidências em `docs/experimentos.md`. Os três cenários foram executados **no mesmo pull
request**, em commits consecutivos:

| Cenário   | Alteração                                       | Resultado                                          | Execução |
|-----------|-------------------------------------------------|----------------------------------------------------|----------|
| Aprovado  | Documentação do experimento                     | 5 gates ✅, Quality Gate ✅                         | _(link)_ |
| Reprovado | `done` passa a poder voltar para `todo` (viola RN07) | Gates 1, 3, 5 ✅; **Gate 2 ❌ (4 testes)**; Gate 4 não executou; **merge bloqueado** | _(link)_ |
| Correção  | `git revert` do commit defeituoso               | 5 gates ✅, Quality Gate ✅                         | _(link)_ |

**Análise.** A alteração defeituosa tinha uma única linha, compilava e passava no lint, ou seja,
seria fácil aprová-la numa revisão apressada. O pipeline bloqueou o merge **na etapa correta** (o
problema era de comportamento, então reprovou o gate de testes) e a regressão foi detectada em três
camadas: domínio (`TestCanTransition`, `TestTransitionTo_Rejected`), serviço
(`TestService_ChangeStatus`) e API (`TestTaskLifecycle`, que esperava `409` e recebeu `200`). O log
mostrou o nome de cada teste e o valor esperado contra o obtido, então a causa pôde ser identificada
sem rodar nada localmente. Tempo da reprovação até a nova aprovação: _(preencher)_.

No experimento complementar, ignorar o erro de `Encode` **não** fez nenhum teste falhar, mas
reprovou o Gate 3 (`errcheck`) com anotação na linha do PR. Isso mostra que os gates são
complementares: cada um pega uma classe diferente de problema. _(manter se executado)_

## 8. Análise crítica

**Quais riscos foram cobertos e quais permaneceram?** R1 a R8 têm verificação automática e
bloqueante. Continuam abertos: **R9 (segredos)**, coberto só por `.gitignore` e revisão, e que
poderia ser automatizado com uma ferramenta como `gitleaks` ou com o *secret scanning* do GitHub;
**desempenho e carga**, sem nenhum teste; **persistência real**, já que o repositório em memória
não exercita banco, migrações nem transações; e **falhas de lógica não especificadas**, porque os
testes verificam as regras que a dupla escreveu, não as que esqueceu.

**O pipeline fornece feedback rápido e compreensível?** _(preencher com a duração real; espera-se
de 1 a 3 minutos)_. A velocidade vem dos gates em paralelo, do cache do Go e de testes que não
dependem de rede nem de banco. A clareza vem da tabela de decisão no resumo, das anotações do lint
na linha exata do PR e dos nomes de teste descritivos em português. Um ponto fraco é que o
`gotestsum` precisa ser instalado a cada execução, o que adiciona alguns segundos.

**Quais verificações continuam manuais?** Revisão de código no PR (legibilidade, regras não
especificadas, segredos), escolha e manutenção dos limites dos gates, avaliação de vulnerabilidades
sem correção disponível e o teste exploratório da API na demonstração.

**Qual é o custo de manter os testes e os gates?** São cerca de 900 linhas de teste para cerca de
720 de código de produção (contando comentários). Toda mudança de regra exige atualizar testes em até três camadas. As
ferramentas fixadas precisam de atualização periódica (Go, linters, *actions*), e cada atualização
pode trazer regras novas. O limite de cobertura pode incentivar testes vazios se não houver revisão.
Em troca, a regressão do experimento, que seria um defeito em produção, custou alguns minutos de CI.

**O que diferencia automação isolada de uma prática DevOps/QAOps?** Ter scripts de teste é
automação. Vira prática QAOps quando: (a) as verificações rodam **sempre**, no fluxo, sem depender
de alguém lembrar; (b) o resultado **decide** se a mudança avança (proteção da branch); (c) as
evidências ficam **registradas** e podem ser auditadas (artefatos e histórico); (d) os critérios
partem dos **riscos** e são revistos pela equipe; e (e) a responsabilidade é **compartilhada**:
nenhum integrante, nem quem configurou o pipeline, consegue contornar o gate.

**O que seria necessário para incluir implantação e observabilidade com segurança?**
(1) Empacotar em imagem de contêiner, com varredura da imagem (ex.: Trivy) como novo gate.
(2) Um job de *deploy* só na `main`, depois do Quality Gate, primeiro em homologação, com aprovação
manual (*environments* do GitHub) antes de produção. (3) Credenciais da nuvem via OIDC ou *Secrets*,
nunca no repositório. (4) *Smoke test* pós-deploy chamando `/health` e um fluxo crítico, com
*rollback* automático se falhar. (5) Observabilidade: os logs já são JSON estruturado (`slog`);
faltam métricas (latência, taxa de erro por rota, ex.: Prometheus), alertas e rastreamento.
(6) Persistência real com testes de integração usando banco em contêiner.

## 9. Contribuições

| Integrante | Partes | Contribuição |
|------------|--------|--------------|
| _(Integrante 1)_ | 1 a 5 | Estrutura do repositório e caracterização do sistema; domínio e regras de negócio; repositório e serviço; API HTTP; testes unitários e de API; pipeline base com os Gates 1 e 2. |
| _(Integrante 2)_ | 6 a 10 | Gate de análise estática (incluindo a correção dos achados); gate de cobertura; gate de dependências e decisão consolidada; proteção da branch; execução e registro dos experimentos; relatório e roteiro da demonstração. |

Os dois integrantes revisaram os PRs um do outro, e o histórico de commits do repositório registra
a autoria de cada parte.

## Referências

- DORA. *DevOps Research and Assessment*. https://dora.dev/
- GitHub. *GitHub Actions documentation*. https://docs.github.com/actions
- GitHub. *About protected branches*. https://docs.github.com/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches
- The Go Authors. *Testing package* e *Data Race Detector*. https://pkg.go.dev/testing ; https://go.dev/doc/articles/race_detector
- The Go Authors. *Vulnerability Management for Go (govulncheck)*. https://go.dev/doc/security/vuln/
- golangci-lint. *Documentation*. https://golangci-lint.run/
- OWASP. *OWASP Top 10: Vulnerable and Outdated Components*. https://owasp.org/
- ISTQB. *Certified Tester Foundation Level (CTFL) v4.0*. https://istqb.org/certifications/certified-tester-foundation-level-ctfl-v4-0/
- Software Testing Material. *QAOps: processo, implementação e benefícios*. https://www.softwaretestingmaterial.com/qaops/
