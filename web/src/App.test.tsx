import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { fakeApi } from './test/fakeApi'

afterEach(() => {
  vi.unstubAllGlobals()
  localStorage.clear()
})

const lane = (name: string) => screen.getByRole('region', { name })

describe('App', () => {
  it('coloca cada tarefa na estação do seu status e mostra o progresso', async () => {
    fakeApi([
      { title: 'Estudar Go', status: 'todo' },
      { title: 'Escrever relatório', status: 'in_progress' },
      { title: 'Entregar trabalho', status: 'done', due_date: '2026-11-05' },
      { title: 'Revisar PRs', status: 'done' },
    ])
    render(<App />)
    await screen.findByText('Estudar Go')

    expect(within(lane('A fazer')).getByText('Estudar Go')).toBeInTheDocument()
    expect(within(lane('Em andamento')).getByText('Escrever relatório')).toBeInTheDocument()
    expect(within(lane('Concluída')).getByText('Entregar trabalho')).toBeInTheDocument()
    expect(within(lane('Concluída')).getByLabelText('2 tarefas')).toBeInTheDocument()
    expect(within(lane('A fazer')).getByText('#001')).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: 'Tarefas concluídas' })).toHaveAttribute('value', '50')
    expect(document.querySelector('time[datetime="2026-11-05"]')).toBeInTheDocument()
  })

  it('mostra as estações vazias quando não há tarefas', async () => {
    fakeApi()
    render(<App />)

    expect(await screen.findAllByText('nada aqui')).toHaveLength(3)
  })

  it('cria uma tarefa e envia o corpo esperado pela API', async () => {
    const fetchMock = fakeApi()
    const user = userEvent.setup()
    render(<App />)

    await user.type(screen.getByLabelText('Título'), 'Revisar pipeline')
    await user.type(screen.getByLabelText('Descrição'), 'Conferir os gates')
    await user.click(screen.getByRole('radio', { name: 'Alta' }))
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    expect(await within(lane('A fazer')).findByText('Revisar pipeline')).toBeInTheDocument()
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')
    expect(JSON.parse(String(post?.[1]?.body))).toEqual({
      title: 'Revisar pipeline',
      description: 'Conferir os gates',
      priority: 'high',
      due_date: null,
    })
    expect(screen.getByLabelText('Título')).toHaveValue('')
    expect(screen.getByRole('radio', { name: 'Média' })).toBeChecked()
    expect(screen.getByRole('status')).toHaveTextContent('Tarefa "Revisar pipeline" entrou na esteira.')
  })

  it('mostra o erro de validação da API e marca o campo inválido (RN01)', async () => {
    fakeApi()
    const user = userEvent.setup()
    render(<App />)

    await user.type(screen.getByLabelText('Título'), 'ab')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Título: deve ter ao menos 3 caracteres')
    expect(screen.getByLabelText('Título')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText('Título')).toHaveValue('ab')
  })

  it('leva a tarefa pelas estações todo → in_progress → done', async () => {
    fakeApi([{ title: 'Escrever relatório', status: 'todo' }])
    const user = userEvent.setup()
    render(<App />)

    const card = await screen.findByRole('listitem', { name: 'Escrever relatório' })
    await user.click(within(card).getByRole('button', { name: 'Iniciar' }))
    const doing = await within(lane('Em andamento')).findByRole('listitem', { name: 'Escrever relatório' })

    await user.click(within(doing).getByRole('button', { name: 'Concluir' }))
    expect(await within(lane('Concluída')).findByRole('listitem', { name: 'Escrever relatório' })).toBeInTheDocument()
    await waitFor(() => expect(within(lane('A fazer')).queryByText('Escrever relatório')).not.toBeInTheDocument())
  })

  it('não oferece transições para tarefa concluída (RN07)', async () => {
    fakeApi([{ title: 'Tarefa encerrada', status: 'done' }])
    render(<App />)

    const card = await screen.findByRole('listitem', { name: 'Tarefa encerrada' })
    const buttons = within(card).getAllByRole('button')
    expect(buttons).toHaveLength(1)
    expect(buttons[0]).toHaveAccessibleName('Excluir Tarefa encerrada')
  })

  it('mostra o conflito quando a API recusa a transição', async () => {
    const fetchMock = fakeApi([{ title: 'Em revisão', status: 'in_progress' }])
    const user = userEvent.setup()
    render(<App />)

    const card = await screen.findByRole('listitem', { name: 'Em revisão' })
    // Simula outra aba que concluiu a tarefa antes deste clique.
    fetchMock.mockImplementationOnce(async () =>
      new Response(JSON.stringify({ error: 'transição de status não permitida' }), { status: 409 }),
    )
    await user.click(within(card).getByRole('button', { name: 'Voltar' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('transição de status não permitida')
    expect(screen.getByText('API conectada')).toBeInTheDocument()
  })

  it('exclui uma tarefa', async () => {
    fakeApi([{ title: 'Apagar isto', status: 'todo' }])
    const user = userEvent.setup()
    render(<App />)

    await user.click(await screen.findByRole('button', { name: 'Excluir Apagar isto' }))

    await waitFor(() => expect(screen.queryByRole('listitem', { name: 'Apagar isto' })).not.toBeInTheDocument())
    expect(screen.getByRole('status')).toHaveTextContent('"Apagar isto" excluída.')
  })

  it('a tecla / leva o foco para o título', async () => {
    fakeApi()
    const user = userEvent.setup()
    render(<App />)
    await screen.findAllByText('nada aqui')

    await user.keyboard('/')
    expect(screen.getByLabelText('Título')).toHaveFocus()
    expect(screen.getByLabelText('Título')).toHaveValue('')
  })

  it('alterna entre modo claro e escuro e lembra a escolha', async () => {
    fakeApi()
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('button', { name: 'Ativar modo escuro' }))
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(localStorage.getItem('esteira-tema')).toBe('dark')

    await user.click(screen.getByRole('button', { name: 'Ativar modo claro' }))
    expect(document.documentElement.dataset.theme).toBe('light')
  })

  it('avisa quando a API está fora do ar', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    render(<App />)

    expect(await screen.findByRole('alert')).toHaveTextContent('Não foi possível falar com a API')
    expect(screen.getByText('API sem conexão')).toBeInTheDocument()
  })
})
