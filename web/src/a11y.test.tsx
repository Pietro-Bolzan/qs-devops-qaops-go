import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import axe from 'axe-core'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { fakeApi } from './test/fakeApi'

// Regras WCAG 2.1 A/AA verificadas pelo axe-core. O contraste de cores não é
// medido aqui (o jsdom não calcula estilos); ele é verificado no navegador real.
async function violations(): Promise<string[]> {
  const result = await axe.run(document.body, {
    runOnly: ['wcag2a', 'wcag2aa', 'wcag21aa'],
    rules: { 'color-contrast': { enabled: false } },
  })
  return result.violations.map((v) => `${v.id}: ${v.help}`)
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('acessibilidade', () => {
  it('a tela com tarefas não tem violações WCAG', async () => {
    fakeApi([
      { title: 'A fazer', status: 'todo', due_date: '2026-11-05', description: 'Com descrição' },
      { title: 'Fazendo', status: 'in_progress', priority: 'high' },
      { title: 'Feita', status: 'done' },
    ])
    render(<App />)
    await screen.findByText('Feita')

    expect(await violations()).toEqual([])
  })

  it('a tela com erro de validação não tem violações WCAG', async () => {
    fakeApi()
    const user = userEvent.setup()
    render(<App />)
    await user.type(screen.getByLabelText('Título'), 'ab')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))
    await screen.findByText(/deve ter ao menos 3 caracteres/)

    expect(await violations()).toEqual([])
  })
})
