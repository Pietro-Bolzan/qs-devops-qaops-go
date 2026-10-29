# Roteiro da demonstração (10 a 12 minutos)

## Preparação (antes da aula)

- [ ] Branch `demo/reabrir-tarefa` criada a partir da `main`, com o patch do experimento já
      aplicado, mas **ainda sem push** (o push é feito ao vivo).
- [ ] Abas abertas: repositório no GitHub, aba **Actions**, PR do experimento já integrado (plano B)
      e `docs/quality-gates.md`.
- [ ] Terminal no diretório do projeto, com `make run` testado, e um segundo terminal com `make web` (interface em http://localhost:5173).
- [ ] Prints das execuções aprovada, reprovada e corrigida (plano B se a rede ou o GitHub falharem).

## Roteiro

| Tempo  | Quem            | O que mostrar |
|--------|-----------------|---------------|
| 0:00–1:30 | Integrante 1 | **Contexto.** O que é a API e a funcionalidade estudada (ciclo de vida da tarefa). Mostrar os riscos R1 a R11 em `docs/sistema.md`: os gates nasceram desses riscos. |
| 1:30–3:00 | Integrante 1 | **Sistema rodando.** `make run` e `make web`: criar uma tarefa na Esteira e levá-la de estação em estação; mostrar que uma tarefa concluída não tem mais botão de transição. No terminal, um `curl` tentando `todo → done`, que devolve `409`. Mostrar rapidamente um teste de tabela (`TestCanTransition`) e um teste de API (`TestTaskLifecycle`). |
| 3:00–5:00 | Integrante 1 | **Pipeline.** Abrir `.github/workflows/ci.yml` e mostrar os gatilhos, os 8 gates, o paralelismo (os Gates 6 e 8 nem esperam o build) e o job `Quality Gate`. Mostrar a regra de proteção da `main` exigindo o check `Quality Gate`. |
| 5:00–6:00 | Integrante 2 | **Acionamento ao vivo.** `git push` da branch `demo/reabrir-tarefa` e abertura do PR. Enquanto roda: explicar a alteração (1 linha que viola a RN07, compila e passa no lint). |
| 6:00–8:00 | Integrante 2 | **Reprovação.** Gate 2 vermelho; abrir o log e mostrar os testes que falharam (domínio, serviço e API) e o "esperado × obtido". Mostrar a tabela "Decisão do Quality Gate" e o botão de merge bloqueado. Baixar o artefato `relatorio-testes`. Se o pipeline demorar, usar o PR já executado. |
| 8:00–9:30 | Integrante 2 | **Correção.** `git revert HEAD` e `git push`, depois a nova execução verde. Mostrar no histórico do PR os três estados: aprovado, reprovado e corrigido. |
| 9:30–11:00 | Integrante 2 | **Evidências e gates.** Mostrar o `coverage.html` (Gate 4), o relatório do `govulncheck` (Gate 5) e uma ficha de gate. Mencionar o achado do Gate 3 (35 `bodyclose` e o falso positivo do `misspell`), a tabela do teste de fumaça (Gate 7) por que o `git revert` não resolve um segredo vazado (Gate 6) e o teste de acessibilidade com axe-core (Gate 8). |
| 11:00–12:00 | Os dois     | **Fechamento.** Como a análise crítica fez o pipeline evoluir (R9 e R10 viraram os Gates 6 e 7), uma limitação que continua (desempenho e persistência real) e o próximo passo (deploy reaproveitando o teste de fumaça, e observabilidade). |

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
