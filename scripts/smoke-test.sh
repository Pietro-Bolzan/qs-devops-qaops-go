#!/usr/bin/env bash
# Teste de fumaça (Gate 7): sobe o BINÁRIO compilado pelo Gate 1, faz chamadas
# HTTP reais ao fluxo principal da API e verifica o desligamento gracioso.
#
# Uso: scripts/smoke-test.sh [caminho-do-binário]   (padrão: bin/tarefas)
#
# Variáveis opcionais:
#   SMOKE_PORT  porta usada no teste (padrão 18080)
#   SMOKE_LOG   arquivo com o log do servidor (padrão reports/smoke-server.log)
#
# Saída: 0 se todas as verificações passam, 1 se alguma falha,
#        2 se o binário não existe ou não é executável.
set -uo pipefail

BIN="${1:-bin/tarefas}"
PORT="${SMOKE_PORT:-18080}"
BASE="http://127.0.0.1:${PORT}"
LOG="${SMOKE_LOG:-reports/smoke-server.log}"

if [ ! -x "$BIN" ]; then
  echo "::error::Teste de fumaça: binário '$BIN' não encontrado ou sem permissão de execução."
  exit 2
fi
mkdir -p "$(dirname "$LOG")"

ADDR="127.0.0.1:${PORT}" "$BIN" >"$LOG" 2>&1 &
PID=$!
BODY="$(mktemp)"
trap 'kill "$PID" 2>/dev/null; rm -f "$BODY"' EXIT

passed=0
failed=0
rows=""

# check <descrição> <esperado> <obtido>
check() {
  if [ "$2" = "$3" ]; then
    passed=$((passed + 1))
    rows+="| $1 | $2 | $3 | ✅ |"$'\n'
    echo "ok    $1 → $3"
  else
    failed=$((failed + 1))
    rows+="| $1 | $2 | $3 | ❌ |"$'\n'
    echo "FALHA $1 → esperado $2, obtido $3"
    echo "::error::Teste de fumaça: '$1' esperava $2 e obteve $3."
  fi
}

# req <método> <caminho> [corpo JSON] → imprime o status HTTP; o corpo fica em $BODY.
req() {
  if [ $# -ge 3 ]; then
    curl -s -o "$BODY" -w '%{http_code}' -X "$1" -H 'Content-Type: application/json' --data "$3" "$BASE$2"
  else
    curl -s -o "$BODY" -w '%{http_code}' -X "$1" "$BASE$2"
  fi
}

# 1. O servidor precisa subir e responder em até 10 s.
ready=0
for _ in $(seq 1 50); do
  if curl -fs "$BASE/health" >/dev/null 2>&1; then
    ready=1
    break
  fi
  kill -0 "$PID" 2>/dev/null || break
  sleep 0.2
done
if [ "$ready" -ne 1 ]; then
  echo "::error::Teste de fumaça: o servidor não respondeu em /health em 10 s."
  echo "--- log do servidor ---"
  cat "$LOG"
  exit 1
fi

# 2. Fluxo principal pela rede, como um cliente real faria.
check "GET /health" 200 "$(req GET /health)"
check "POST /tasks válida" 201 "$(req POST /tasks '{"title":"Teste de fumaça","priority":"high"}')"
ID="$(sed -nE 's/.*"id":([0-9]+).*/\1/p' "$BODY")"
check "resposta traz o id da tarefa" "sim" "$([ -n "$ID" ] && echo sim || echo não)"
ID="${ID:-0}"
check "POST /tasks sem título (RN01)" 422 "$(req POST /tasks '{"title":""}')"
check "GET /tasks/{id}" 200 "$(req GET "/tasks/$ID")"
check "PATCH status → in_progress" 200 "$(req PATCH "/tasks/$ID/status" '{"status":"in_progress"}')"
check "PATCH status → done" 200 "$(req PATCH "/tasks/$ID/status" '{"status":"done"}')"
check "PATCH status done → todo (RN07)" 409 "$(req PATCH "/tasks/$ID/status" '{"status":"todo"}')"
check "GET /tasks/999999 inexistente" 404 "$(req GET /tasks/999999)"

# 3. Desligamento gracioso: com SIGTERM o processo deve terminar com código 0.
kill -TERM "$PID"
wait "$PID"
check "SIGTERM encerra com código 0" 0 "$?"

total=$((passed + failed))
echo "Teste de fumaça: $passed de $total verificações aprovadas."

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    echo "## Gate 7 · Teste de fumaça"
    echo "| Verificação | Esperado | Obtido | Resultado |"
    echo "|-------------|---------:|-------:|:---------:|"
    printf '%s' "$rows"
    echo
    echo "**$passed de $total verificações aprovadas.**"
  } >>"$GITHUB_STEP_SUMMARY"
fi

[ "$failed" -eq 0 ]
