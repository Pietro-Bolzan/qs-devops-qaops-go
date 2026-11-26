# Monitoramento de erros (Sentry)

O pipeline impede que código com defeito **conhecido** chegue à `main`. Os defeitos que escapam dos
testes só aparecem com a API rodando. O monitoramento de erros fecha esse ciclo de feedback: quando
a API devolve um erro 500 ou um handler entra em *panic*, o evento vai para o painel do
[Sentry](https://sentry.io) com a rota, a pilha de chamadas, o ambiente e a versão do binário.

## Por que Sentry

| Opção avaliada | Custo | Esforço | Decisão |
|----------------|-------|---------|---------|
| **Sentry** (plano Developer) | Grátis: 1 usuário e 5 mil erros/mês | Uma variável de ambiente | **Escolhido**: captura o erro com contexto e agrupa ocorrências iguais. |
| Grafana + Prometheus | Grátis, mas exige servidor próprio ou Grafana Cloud | Exportar métricas, subir e manter os serviços, montar painéis | **Adiado** (Stand by no Kanban): mostra *quantos* erros, não *qual* erro. Faz sentido junto com o *deploy*. |
| SonarCloud | Grátis para repositório público | Integração no CI | **Não se aplica aqui**: é análise estática, papel que o Gate 3 (golangci-lint) já cumpre. |

## Como funciona

O pacote [`internal/monitor`](../internal/monitor/monitor.go) usa **só a biblioteca padrão do Go**:
monta o envelope do protocolo do Sentry e o envia por HTTPS. Não usamos o SDK oficial para manter o
módulo sem dependências externas, o que mantém o Gate 5 (dependências) sem superfície nova de ataque.

```text
requisição ──► Middleware (recupera panic) ──► handlers da API
                    │                              │
                    │ panic                        │ erro inesperado (500)
                    ▼                              ▼
              monitor.Reporter ── fila (até 100 eventos) ──► envio em segundo plano ──► Sentry
```

| Situação | O que acontece |
|----------|----------------|
| Erro de domínio (400, 404, 409, 422) | **Não** vai ao Sentry: é erro do cliente, não defeito. |
| Erro inesperado (500) | Vai ao Sentry com nível `error`, a rota e a pilha. O cliente recebe só `erro interno`. |
| *Panic* em um handler | O middleware recupera, responde 500 em JSON e envia o evento com nível `fatal`. |
| `SENTRY_DSN` vazia | O monitoramento fica desligado e a API funciona igual. É o caso dos testes e do CI. |
| Sentry fora do ar ou lento | O envio falha com tempo limite de 5 s e só gera um aviso no log. A API não trava. |
| Muitos erros de uma vez | Eventos acima de 100 na fila são descartados para não consumir memória. |
| Desligamento do servidor | Os eventos pendentes são enviados antes de sair (até 5 s). |

Para proteger dados pessoais, o evento leva só o **método e o caminho** da requisição (sem query
string, corpo ou cabeçalhos).

## Configuração

| Variável | Obrigatória | Descrição |
|----------|-------------|-----------|
| `SENTRY_DSN` | Não | DSN do projeto no Sentry (*Settings → Projects → Client Keys*). Vazia desliga o envio. |
| `SENTRY_ENVIRONMENT` | Não | Nome do ambiente nos eventos. Padrão: `local`. |

1. Crie uma conta gratuita em <https://sentry.io> e um projeto da plataforma **Go**.
2. Copie a DSN do projeto.
3. Rode a API com a DSN no ambiente (nunca no código):

   ```bash
   SENTRY_DSN="https://<chave>@<org>.ingest.sentry.io/<projeto>" make run
   ```

4. Confira a integração enviando um evento de teste:

   ```bash
   SENTRY_DSN="https://<chave>@<org>.ingest.sentry.io/<projeto>" make sentry-check
   ```

   O evento aparece em **Issues** no painel do Sentry.

Para um ambiente publicado, a DSN fica nos **Secrets** da plataforma (no GitHub: *Settings → Secrets
and variables → Actions → `SENTRY_DSN`*). A DSN não dá acesso de leitura aos dados do projeto, só
permite enviar eventos, mas mesmo assim não deve ser versionada: o Gate 6 (gitleaks) reprova o commit.

## Evidências e testes

- [`internal/monitor/monitor_test.go`](../internal/monitor/monitor_test.go) sobe um Sentry falso com
  `httptest` e verifica o formato do envelope, a autenticação, a omissão da query string, o
  comportamento com o Sentry fora do ar, a fila cheia e o prazo do desligamento.
- [`internal/httpapi/handler_test.go`](../internal/httpapi/handler_test.go)
  (`TestInternalErrorIsReported`) garante que só os erros 500 são reportados, e os 4xx não.
- Os testes não usam rede externa nem conta no Sentry, então o pacote entra no Gate 2 (testes) e no
  Gate 4 (cobertura) como qualquer outro.

## Limitações

- Não mede desempenho nem latência: o Sentry gratuito só é usado aqui para **erros**.
- Não há alertas configurados. No painel do Sentry dá para criar uma regra de alerta por e-mail
  quando um erro novo aparece.
- O agrupamento usa o tipo do erro mais interno e a pilha; erros diferentes com o mesmo tipo podem
  cair no mesmo grupo.
- Como a API ainda não é implantada, o monitoramento só funciona em execuções locais com a DSN
  configurada. O valor completo aparece junto com o *deploy* (ver a análise crítica do relatório).
