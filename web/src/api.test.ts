import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api } from './api'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('api', () => {
  it('usa o erro e o campo informados pela API', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'é obrigatório', field: 'title' }), { status: 422 })),
    )

    const err = await api.create({ title: '', description: '', priority: 'low', due_date: null }).catch((e) => e)

    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 422, message: 'é obrigatório', field: 'title' })
  })

  it('gera mensagem genérica quando o erro não vem em JSON', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('falha', { status: 502 })))

    await expect(api.list()).rejects.toMatchObject({ status: 502, message: 'Erro HTTP 502' })
  })

  it('filtra por status na própria consulta', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('[]', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.list('done')).resolves.toEqual([])
    expect(fetchMock).toHaveBeenCalledWith('/api/tasks?status=done', { headers: undefined })
  })

  it('aceita 204 sem corpo na exclusão', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.remove(7)).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledWith('/api/tasks/7', { method: 'DELETE', headers: undefined })
  })

  it('envia JSON com cabeçalho na mudança de status', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.changeStatus(3, 'done')

    expect(fetchMock).toHaveBeenCalledWith('/api/tasks/3/status', {
      method: 'PATCH',
      body: '{"status":"done"}',
      headers: { 'Content-Type': 'application/json' },
    })
  })
})
