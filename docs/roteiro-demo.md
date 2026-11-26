# Roteiro da demonstração (10 a 12 minutos)

## Preparação (antes da aula)

- [ ] Branch `demo/reabrir-tarefa` criada a partir da `main`, com o patch do experimento já
      aplicado, mas **ainda sem push** (o push é feito ao vivo).
- [ ] Abas abertas: repositório no GitHub, aba **Actions**, PR do experimento já integrado (plano B),
      `docs/quality-gates.md`, o **Kanban** (GitHub Projects) e o painel **Issues** do Sentry.
- [ ] `export SENTRY_DSN=...` feito no terminal da API (a DSN fica só no terminal, nunca no repositório)
      e `make sentry-check` testado uma vez antes da aula.
- [ ] Terminal no diretório do projeto, com `make run` testado, e um segundo terminal com `make web` (interface em http://localhost:5173).
- [ ] Prints das execuções aprovada, reprovada e corrigida (plano B se a rede ou o GitHub falharem).

## Roteiro

| Tempo  | Quem            | O que mostrar |
|--------|-----------------|---------------|
| 0:00–1:30 | Integrante 1 | **Contexto.** O que é a API e a funcionalidade estudada (ciclo de vida da tarefa). Mostrar os riscos R1 a R11 em `docs/sistema.md`: os gates nasceram desses riscos. Mostrar rapidamente o **Kanban**: cada parte passou por Aprovado → Em andamento → Controle de qualidade (PR com os gates) → Concluído (merge). |
| 1:30–3:00 | Integrante 1 | **Sistema rodando.** `make run` e `make web`: criar uma tarefa na Esteira e levá-la de estação em estação; mostrar que uma tarefa concluída não tem mais botão de transição. No terminal, um `curl` tentando `todo → done`, que devolve `409`. Mostrar rapidamente um teste de tabela (`TestCanTransition`) e um teste de API (`TestTaskLifecycle`). |
| 3:00–5:00 | Integrante 1 | **Pipeline.** Abrir `.github/workflows/ci.yml` e mostrar os gatilhos, os 8 gates, o paralelismo (os Gates 6 e 8 nem esperam o build) e o job `Quality Gate`. Mostrar a regra de proteção da `main` exigindo o check `Quality Gate`. |
| 5:00–6:00 | Integrante 2 | **Acionamento.** Mostrar no PR #16 que cada push disparou o pipeline sozinho (execuções #33, #34 e #35). Explicar a alteração: 1 linha que viola a RN07, compila e passa no lint. Se o professor pedir ao vivo: `git push` da branch `demo/reabrir-tarefa` e abertura do PR. |
| 6:00–8:00 | Integrante 2 | **Reprovação.** Na execução #34: Gates 2 e 7 vermelhos. Abrir o log do Gate 2 (falhas no domínio, no serviço e na API) e a tabela do teste de fumaça (RN07: esperado 409, obtido 200). Mostrar a tabela "Decisão do Quality Gate" e o merge bloqueado. Baixar o artefato `relatorio-testes`. |
| 8:00–9:30 | Integrante 2 | **Correção.** Mostrar a execução #35, verde depois do `git revert`, e no histórico do PR os três estados: aprovado, reprovado e corrigido. |
| 9:30–10:30 | Integrante 2 | **Evidências e gates.** Mostrar o `coverage.html` (Gate 4), o relatório do `govulncheck` (Gate 5) e uma ficha de gate. Mencionar o achado do Gate 3 (35 `bodyclose` e o falso positivo do `misspell`), a tabela do teste de fumaça (Gate 7), por que o `git revert` não resolve um segredo vazado (Gate 6) e o teste de acessibilidade com axe-core (Gate 8). |
| 10:30–11:15 | Integrante 2 | **Monitoramento (Sentry).** O pipeline barra defeitos conhecidos; os que escapam aparecem em execução. Rodar `make sentry-check` e mostrar o evento chegando em **Issues** no Sentry (ambiente, versão e pilha). Explicar: só erros 500 e *panics* vão ao Sentry, 4xx não; sem `SENTRY_DSN` fica desligado, por isso os testes e o CI não dependem dele. |
| 11:15–12:00 | Os dois     | **Fechamento.** Como a análise crítica fez o pipeline evoluir (R9 e R10 viraram os Gates 6 e 7), uma limitação que continua (desempenho e persistência real) e o próximo passo (deploy reaproveitando o teste de fumaça, métricas com Grafana). |

## Comandos usados ao vivo

```bash
make run
```

```bash
curl -X POST localhost:8080/tasks -d '{"title":"Apresentar trabalho","priority":"high"}'
```

```bash
curl -X PATCH localhost:8080/tasks/1/status -d '{"status":"done"}'
```

```bash
git push -u origin demo/reabrir-tarefa
```

```bash
make sentry-check   # com SENTRY_DSN exportada no terminal
```

```bash
git revert --no-edit HEAD && git push
```

## Perguntas prováveis (cada integrante deve saber responder)

- Por que o limite de cobertura é 90% no domínio e não 100%?
- Por que o experimento reprovou no Gate 2 e não no Gate 1 ou no Gate 3?
- O que acontece com o Gate 4 quando o Gate 2 falha? Por quê?
- Por que existe o job `Quality Gate` separado? O que aconteceria sem a proteção da branch?
- O que o `govulncheck` faz de diferente de uma lista de versões vulneráveis?
- Por que os testes usam um relógio fixo?
- O que o `-race` detecta? Mostre um teste que depende dele.
- Qual risco continua sem verificação automática? (desempenho/carga e persistência real)
- Por que o Gate 6 analisa todo o histórico e não só os arquivos atuais?
- Qual a diferença entre o teste de fumaça (Gate 7) e os testes de API (Gate 2)?
- Como a acessibilidade virou um critério objetivo de quality gate? O que o jsdom não consegue medir?
- Qual a diferença entre ter testes automatizados e praticar QAOps?
- Por que o Sentry recebe só erros 500 e não 404 ou 422? Por que não usamos o SDK oficial?
- Por que o Sentry e não o Grafana? O que o Grafana mostraria que o Sentry não mostra?
- Onde fica a `SENTRY_DSN`? O que acontece se ela for parar no repositório?
- O que significa cada coluna do Kanban? Quando um cartão sai de Controle de qualidade?
