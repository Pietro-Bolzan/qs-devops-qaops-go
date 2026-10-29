#!/usr/bin/env bash
# Gate 4 · Cobertura de testes.
#
# Lê um perfil de cobertura do Go (go test -coverprofile) e reprova se a
# cobertura de comandos ficar abaixo dos limites:
#   - domínio (internal/task):      COVERAGE_MIN_DOMAIN (padrão 90%)
#   - total, sem cmd/ (internal/*): COVERAGE_MIN_TOTAL  (padrão 80%)
#
# Uso: scripts/coverage-gate.sh [reports/coverage.out]
set -euo pipefail

PROFILE="${1:-reports/coverage.out}"
MIN_TOTAL="${COVERAGE_MIN_TOTAL:-80}"
MIN_DOMAIN="${COVERAGE_MIN_DOMAIN:-90}"
DOMAIN_PATH="/internal/task/"
EXCLUDE_PATH="/cmd/"

if [[ ! -f "$PROFILE" ]]; then
  echo "ERRO: perfil de cobertura '$PROFILE' não encontrado. Rode os testes com -coverprofile." >&2
  exit 2
fi

# coverage <filtro> → percentual de comandos cobertos (uma casa decimal).
# Linhas do perfil: "arquivo.go:linha.col,linha.col <comandos> <execuções>".
# Blocos repetidos (mesmo trecho em mais de um pacote) são contados uma vez.
coverage() {
  awk -v scope="$1" -v skip="$EXCLUDE_PATH" '
    NR == 1 { next }
    index($1, scope) == 0 || index($1, skip) > 0 { next }
    {
      stmts[$1] = $2
      if ($3 > 0) hit[$1] = 1
    }
    END {
      for (b in stmts) { total += stmts[b]; if (b in hit) covered += stmts[b] }
      if (total == 0) { print "0.0"; exit }
      printf "%.1f", covered * 100 / total
    }' "$PROFILE"
}

# below <valor> <mínimo> → verdadeiro se valor < mínimo.
below() { awk -v v="$1" -v m="$2" 'BEGIN { exit !(v + 0 < m + 0) }'; }

domain=$(coverage "$DOMAIN_PATH")
total=$(coverage "/internal/")

status() { if below "$1" "$2"; then echo "REPROVADO"; else echo "aprovado"; fi; }
domain_status=$(status "$domain" "$MIN_DOMAIN")
total_status=$(status "$total" "$MIN_TOTAL")

failed=0
if below "$domain" "$MIN_DOMAIN" || below "$total" "$MIN_TOTAL"; then failed=1; fi

report=$(cat <<EOF
## Gate 4 · Cobertura

| Escopo                       | Cobertura | Mínimo | Resultado |
|------------------------------|----------:|-------:|-----------|
| Domínio (\`internal/task\`)  | ${domain}% | ${MIN_DOMAIN}% | ${domain_status} |
| Total (\`internal/...\`)     | ${total}% | ${MIN_TOTAL}% | ${total_status} |
EOF
)

echo "$report"
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  echo "$report" >> "$GITHUB_STEP_SUMMARY"
fi

if [[ "$failed" -ne 0 ]]; then
  echo "::error::Cobertura abaixo do mínimo definido. Adicione testes para os trechos não cobertos (veja coverage.html)."
  exit 1
fi
echo "Cobertura dentro dos limites."
