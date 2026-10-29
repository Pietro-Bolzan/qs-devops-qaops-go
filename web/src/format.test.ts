import { describe, expect, it } from 'vitest'
import { formatDue, relativeDue, ticket } from './format'

const today = new Date(2026, 9, 29) // 29/10/2026

describe('format', () => {
  it('formata o prazo sem deslocar o dia por causa do fuso', () => {
    expect(formatDue('2026-11-05')).toBe('05 de nov')
  })

  it.each([
    ['2026-10-27', 'atrasada 2 dias', 'late'],
    ['2026-10-28', 'venceu ontem', 'late'],
    ['2026-10-29', 'vence hoje', 'soon'],
    ['2026-10-30', 'vence amanhã', 'soon'],
    ['2026-11-01', 'em 3 dias', 'soon'],
    ['2026-11-10', 'em 12 dias', 'later'],
  ])('prazo %s → %s', (iso, label, tone) => {
    expect(relativeDue(iso, today)).toEqual({ label, tone })
  })

  it('numera o ticket com três dígitos', () => {
    expect(ticket(7)).toBe('#007')
    expect(ticket(1234)).toBe('#1234')
  })
})
