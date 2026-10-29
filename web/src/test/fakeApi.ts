import { vi } from 'vitest'
import type { Status, Task } from '../api'

// fakeApi substitui o fetch por uma versão em memória que imita a API Go:
// valida o título (RN01) e recusa transições fora do ciclo (RN06/RN07).
export function fakeApi(initial: Partial<Task>[] = []) {
  let nextId = 1
  const now = '2026-10-29T12:00:00Z'
  let tasks: Task[] = initial.map((t) => ({
    id: nextId++,
    title: 'Tarefa',
    priority: 'medium',
    status: 'todo',
    created_at: now,
    updated_at: now,
    ...t,
  }))

  const allowed: Record<Status, Status[]> = { todo: ['in_progress'], in_progress: ['todo', 'done'], done: [] }
  const json = (status: number, body: unknown) =>
    new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

  const fetchMock = vi.fn(async (input: string, init: RequestInit = {}) => {
    const url = new URL(input, 'http://localhost')
    const method = init.method ?? 'GET'
    const body = init.body ? JSON.parse(String(init.body)) : {}
    const match = url.pathname.match(/^\/api\/tasks\/(\d+)/)
    const task = match ? tasks.find((t) => t.id === Number(match[1])) : undefined

    if (method === 'GET' && url.pathname === '/api/tasks') {
      const status = url.searchParams.get('status')
      return json(200, status ? tasks.filter((t) => t.status === status) : tasks)
    }
    if (method === 'POST' && url.pathname === '/api/tasks') {
      if (String(body.title ?? '').trim().length < 3) {
        return json(422, { error: 'deve ter ao menos 3 caracteres', field: 'title' })
      }
      const created: Task = { id: nextId++, status: 'todo', created_at: now, updated_at: now, ...body }
      tasks = [...tasks, created]
      return json(201, created)
    }
    if (!task) {
      return json(404, { error: 'tarefa não encontrada' })
    }
    if (method === 'PATCH') {
      if (!allowed[task.status].includes(body.status)) {
        return json(409, { error: 'transição de status não permitida' })
      }
      task.status = body.status
      return json(200, task)
    }
    if (method === 'DELETE') {
      tasks = tasks.filter((t) => t.id !== task.id)
      return new Response(null, { status: 204 })
    }
    return json(405, { error: 'método não permitido' })
  })

  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}
