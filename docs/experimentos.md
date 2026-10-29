# Experimentos: cenários aprovado, reprovado e corrigido

Objetivo: mostrar que o pipeline **detecta uma alteração inadequada e impede que ela avance**, que
a falha aparece **na etapa certa** e que, depois da correção, o fluxo volta a ser aprovado,
**com o histórico preservado**.

> Pré-requisito: a proteção da branch `main` com o check obrigatório `Quality Gate` (ver
> [quality-gates.md](quality-gates.md#proteção-da-branch-main)). Sem ela o pipeline fica vermelho,
> mas o botão de merge continua liberado.

## Desenho do experimento

Os três cenários acontecem **no mesmo pull request**, em commits consecutivos. Assim a aba
*Checks* e o histórico do PR mostram a sequência verde → vermelho → verde lado a lado.

| # | Cenário    | Commit no PR                                                   | Resultado esperado                                                   |
|---|------------|----------------------------------------------------------------|----------------------------------------------------------------------|
| 1 | Aprovado   | `docs: adiciona roteiro e registro dos experimentos`           | Os 5 gates e o Quality Gate aprovados                                |
| 2 | Reprovado  | `feat(task): permite reabrir tarefa concluída`                 | Gates 1, 3 e 5 aprovados; **Gate 2 reprovado**; Gate 4 não executa; **Quality Gate bloqueia o merge** |
| 3 | Correção   | `Revert "feat(task): permite reabrir tarefa concluída"`        | Os 5 gates e o Quality Gate aprovados de novo                        |

### Alteração controlada (cenário 2)

Uma pessoa "simplifica" o ciclo de vida para permitir reabrir tarefas concluídas, mudando uma
única linha em `internal/task/task.go`:

```diff
-	StatusDone:       {},
+	StatusDone:       {StatusTodo}, // permite reabrir tarefa concluída
```

A mudança **compila**, **passa no lint** e **não usa nenhuma dependência nova**, mas viola a
regra de negócio RN07 ("tarefa `done` é final"). É o tipo de defeito que só os testes pegam e que
seria fácil aprovar numa revisão apressada, por isso foi escolhido.

**Por que a falha aparece na etapa certa:** o problema é de **comportamento**, então quem deve
reprovar é o Gate 2 (testes), e não o build ou o lint.

Testes que devem falhar (validado localmente antes do experimento):

| Teste                                   | Nível        | Por que falha                                     |
|-----------------------------------------|--------------|---------------------------------------------------|
| `TestCanTransition`                     | Unitário     | `done → todo` passou a ser permitida              |
| `TestTransitionTo_Rejected`             | Unitário     | Tarefa concluída foi reaberta                     |
| `TestService_ChangeStatus`              | Unitário     | Serviço aceitou `done → todo`                     |
| `TestTaskLifecycle`                     | API/integração | `PATCH /tasks/1/status` devolveu 200 em vez de 409 |

A regressão foi detectada em **três camadas** (domínio, serviço e API): mesmo que um teste fosse
removido, os outros ainda bloqueariam.

### Correção (cenário 3)

A causa é desfeita com `git revert`, o que **preserva o histórico**: o commit com defeito e o commit
que o desfaz continuam visíveis. O PR é integrado com **"Create a merge commit"** (e não com
*squash*) para que os três commits cheguem à `main`.

## Registro das execuções

Execuções do PR [#9](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/pull/9), em 29/10/2026
(horários em UTC). A duração vai da criação da execução até a decisão do Quality Gate.

| Cenário   | Commit (hash) | Execução (link)                     | Início   | Duração total | Gate que reprovou | Decisão      |
|-----------|---------------|-------------------------------------|----------|---------------|-------------------|--------------|
| Aprovado  | `ac131ef`     | [40879768649](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/actions/runs/40879768649) | 13:12:09 | 1min51s       | -                 | ✅ Aprovado  |
| Reprovado | `7fc8c36`     | [40880227916](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/actions/runs/40880227916) | 13:15:10 | 1min09s       | Gate 2 · Testes   | ❌ Bloqueado |
| Correção  | `dd1394a`     | [40882005466](https://github.com/Pietro-Bolzan/qs-devops-qaops-go/actions/runs/40882005466) | 13:26:52 | 1min37s       | -                 | ✅ Aprovado  |

Tempo de cada gate (em segundos):

| Cenário   | Build | Testes | Análise estática | Cobertura | Dependências | Quality Gate |
|-----------|------:|-------:|-----------------:|----------:|-------------:|-------------:|
| Aprovado  | 29    | 47     | 23               | 20        | 27           | 4            |
| Reprovado | 16    | 41 ❌  | 17               | —         | 25           | 4 ❌         |
| Correção  | 15    | 47     | 22               | 19        | 28           | 4            |

### O que a execução reprovada mostrou

Resumo do Gate 2: **72 testes, 5 falhas, 0 erros**. As 5 falhas correspondem às **4 funções de
teste previstas** (o subteste "tarefa concluída não pode ser reaberta" conta separado do teste pai):

```text
=== FAIL: internal/httpapi TestTaskLifecycle
    handler_test.go:300: PATCH /tasks/1/status: status = 200, esperado 409
=== FAIL: internal/task TestService_ChangeStatus
    service_test.go:141: done → todo deveria ser rejeitada, obteve <nil>
=== FAIL: internal/task TestCanTransition
    task_test.go:117: CanTransition(done, todo) = true, esperado false
=== FAIL: internal/task TestTransitionTo_Rejected/tarefa_concluída_não_pode_ser_reaberta
    task_test.go:180: done → todo deveria ser rejeitada, obteve <nil>
DONE 72 tests, 5 failures
```

A tabela "Decisão do Quality Gate" registrou: Build ✅, Testes ❌, Análise estática ✅, Cobertura ⏭️
(não executado), Dependências ✅ → **Decisão: ❌ BLOQUEADO**. No PR, o check obrigatório
`Quality Gate` ficou vermelho e o botão de merge, desabilitado.

Evidências a capturar (prints para o relatório e a apresentação):

- [x] Lista de checks do PR com o Gate 2 vermelho e o merge bloqueado.
- [x] Log do Gate 2 mostrando os nomes dos testes que falharam e a mensagem de erro.
- [x] Tabela "Decisão do Quality Gate" no resumo da execução reprovada.
- [x] Artefato `relatorio-testes` da execução reprovada (`junit.xml`), disponível por 30 dias na execução 40880227916.
- [x] Lista de checks verde depois do revert.
- [x] Histórico de commits do PR mostrando os três commits (✅ → ❌ → ✅).

## Comparação dos resultados

| Aspecto                         | Aprovado | Reprovado                             | Correção |
|---------------------------------|----------|---------------------------------------|----------|
| Gate 1 · Build                  | ✅       | ✅                                    | ✅       |
| Gate 2 · Testes                 | ✅       | ❌ 4 testes falharam                  | ✅       |
| Gate 3 · Análise estática       | ✅       | ✅                                    | ✅       |
| Gate 4 · Cobertura              | ✅       | ⏭️ não executou (depende do Gate 2)   | ✅       |
| Gate 5 · Dependências           | ✅       | ✅                                    | ✅       |
| Quality Gate                    | ✅       | ❌ merge bloqueado                    | ✅       |
| Tempo até o feedback            | 1min51s  | 1min09s                               | 1min37s  |

**Leitura:** em **1min09s** depois do push, o PR mostrou que a regra RN07 tinha sido violada, em
quais testes e em quais camadas (domínio, serviço e API), sem ninguém precisar executar nada
localmente. A reprovação foi **mais rápida** que as aprovações porque o Gate 4 não precisou rodar e
o Quality Gate decidiu assim que o Gate 2 falhou. O build, o lint e a análise de dependências
passaram: a alteração era "válida" para essas ferramentas, e só os testes de comportamento a
barraram, ou seja, a falha apareceu **na etapa certa**. A correção com `git revert` voltou a ser
aprovada em **1min37s**, e entre o fim da execução reprovada (13:16:19) e a nova aprovação
(13:28:29) passaram cerca de 12 minutos, a maior parte para analisar o log e registrar as
evidências. O histórico do PR preserva os três commits.

## Experimento complementar (opcional): falha de análise estática

Mostra que um gate diferente reprova quando o problema é de **qualidade de código** e não de
comportamento. Em `internal/httpapi/handler.go`, o erro de escrita da resposta passa a ser ignorado:

```diff
-	if err := json.NewEncoder(w).Encode(body); err != nil {
-		h.log.Error("falha ao escrever resposta", "erro", err)
-	}
+	json.NewEncoder(w).Encode(body)
```

Resultado esperado (validado localmente): build ✅, **testes ✅** (o comportamento não muda),
**Gate 3 ❌** com `Error return value of (*encoding/json.Encoder).Encode is not checked (errcheck)`,
anotado na própria linha do PR. Isso mostra por que os testes sozinhos não bastam.
