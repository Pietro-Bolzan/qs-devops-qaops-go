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

> Preencher com os dados reais das execuções no GitHub Actions (aba **Actions** → execução →
> copiar a URL; a duração aparece no topo da página).

| Cenário   | Commit (hash) | Execução (link)     | Duração total | Gate que reprovou | Decisão      |
|-----------|---------------|---------------------|---------------|-------------------|--------------|
| Aprovado  | `_______`     | _link_              | _mm:ss_       | -                 | ✅ Aprovado  |
| Reprovado | `_______`     | _link_              | _mm:ss_       | Gate 2 · Testes   | ❌ Bloqueado |
| Correção  | `_______`     | _link_              | _mm:ss_       | -                 | ✅ Aprovado  |

Evidências a capturar (prints para o relatório e a apresentação):

- [ ] Lista de checks do PR com o Gate 2 vermelho e a mensagem "Merging is blocked".
- [ ] Log do Gate 2 mostrando os nomes dos testes que falharam e a mensagem de erro.
- [ ] Tabela "Decisão do Quality Gate" no resumo da execução reprovada.
- [ ] Artefato `relatorio-testes` baixado da execução reprovada (`junit.xml`).
- [ ] Lista de checks verde depois do revert.
- [ ] Histórico de commits do PR mostrando os três commits.

## Comparação dos resultados

| Aspecto                         | Aprovado | Reprovado                             | Correção |
|---------------------------------|----------|---------------------------------------|----------|
| Gate 1 · Build                  | ✅       | ✅                                    | ✅       |
| Gate 2 · Testes                 | ✅       | ❌ 4 testes falharam                  | ✅       |
| Gate 3 · Análise estática       | ✅       | ✅                                    | ✅       |
| Gate 4 · Cobertura              | ✅       | ⏭️ não executou (depende do Gate 2)   | ✅       |
| Gate 5 · Dependências           | ✅       | ✅                                    | ✅       |
| Quality Gate                    | ✅       | ❌ merge bloqueado                    | ✅       |
| Tempo até o feedback            | _mm:ss_  | _mm:ss_                               | _mm:ss_  |

**Leitura:** _(preencher após executar)_ Ex.: em _N_ minutos o PR mostrou qual regra foi violada e
em quais testes, sem precisar executar nada localmente. A correção levou _N_ minutos entre o
revert e a nova aprovação.

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
