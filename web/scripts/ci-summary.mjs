// Escreve no resumo da execução do GitHub Actions a tabela do Gate 8
// (testes e cobertura do frontend). Uso: node scripts/ci-summary.mjs
import { appendFileSync, existsSync, readFileSync } from 'node:fs'

const lines = ['## Gate 8 · Frontend']

if (existsSync('reports/junit.xml')) {
  const header = readFileSync('reports/junit.xml', 'utf8').match(/<testsuites[^>]*>/)?.[0] ?? ''
  const attr = (name) => header.match(new RegExp(`${name}="([^"]*)"`))?.[1] ?? '?'
  lines.push('| Testes | Falhas | Erros | Tempo (s) |', '|-------:|-------:|------:|----------:|')
  lines.push(`| ${attr('tests')} | ${attr('failures')} | ${attr('errors')} | ${attr('time')} |`)
} else {
  lines.push('Relatório JUnit não foi gerado.')
}

if (existsSync('reports/coverage/coverage-summary.json')) {
  const { total } = JSON.parse(readFileSync('reports/coverage/coverage-summary.json', 'utf8'))
  lines.push('', '| Cobertura (`web/src`) | Linhas | Ramos | Funções | Mínimo |', '|---|---:|---:|---:|---:|')
  lines.push(`| total | ${total.lines.pct}% | ${total.branches.pct}% | ${total.functions.pct}% | 80% |`)
}

const output = `${lines.join('\n')}\n`
if (process.env.GITHUB_STEP_SUMMARY) {
  appendFileSync(process.env.GITHUB_STEP_SUMMARY, output)
}
console.log(output)
