# Reexecução do experimento com o pipeline completo (8 gates)

O experimento oficial (PR #9, ver [experimentos.md](experimentos.md)) foi executado quando o
pipeline tinha 5 gates. Depois dele entraram os Gates 6 (segredos), 7 (teste de fumaça) e
8 (frontend), além do monitoramento de erros. Esta reexecução repete os três cenários com o
pipeline completo, para confirmar que a decisão continua correta com todos os gates.

| Cenário   | Alteração                                                        | Resultado esperado |
|-----------|------------------------------------------------------------------|--------------------|
| Aprovado  | Este documento (sem mudança de código)                           | 8 gates aprovados e Quality Gate aprovado |
| Reprovado | `done` passa a poder voltar para `todo`, violando a RN07         | Gate 2 reprovado, Gate 4 não executado, merge bloqueado |
| Correção  | `git revert` do commit defeituoso                                | 8 gates aprovados de novo, com o histórico preservado |

A alteração defeituosa é a mesma do experimento original (commit `7fc8c36`): uma linha que
compila e passa na análise estática, mas quebra o comportamento. Os resultados ficam
registrados nas execuções do PR #16.

## Resultados

| Cenário   | Commit    | Execução | Resultado | Duração |
|-----------|-----------|----------|-----------|---------|
| Aprovado  | `f7d3b2d` | [#33](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/actions/runs/45881094543) | 8 gates ✅, Quality Gate ✅ | 1min19s |
| Reprovado | `5b942aa` | [#34](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/actions/runs/45881377618) | Gate 2 ❌ (90 testes, 5 falhas), Gate 7 ❌ (9 de 10), Gate 4 não executado, demais ✅, Quality Gate ❌ BLOQUEADO | 1min19s |
| Correção  | `f651af3` | [#35](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/actions/runs/45881774448) | 8 gates ✅, Quality Gate ✅ | 1min20s |

**Falhas do Gate 2 (execução #34):**

- `TestCanTransition`: `CanTransition(done, todo) = true, esperado false`
- `TestTransitionTo_Rejected/tarefa_concluída_não_pode_ser_reaberta`: `done → todo deveria ser rejeitada`
- `TestService_ChangeStatus`: `done → todo deveria ser rejeitada, obteve <nil>`
- `TestTaskLifecycle`: `PATCH /tasks/1/status: status = 200, esperado 409`

**Falha do Gate 7 (execução #34):** `PATCH status done → todo (RN07)`: esperado 409, obtido 200
(9 de 10 verificações aprovadas).

**Comparação com o PR #9.** A decisão foi a mesma (merge bloqueado), mas agora dois gates
independentes reprovaram: os testes em memória (Gate 2) e o binário real pela rede (Gate 7), que
ainda não existia na primeira rodada.
