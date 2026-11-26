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
registrados nas execuções deste PR.
