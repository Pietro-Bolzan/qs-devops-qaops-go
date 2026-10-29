// Cliente HTTP da API de Tarefas. Em desenvolvimento, o Vite encaminha
// /api para o servidor Go (veja vite.config.ts), então o navegador fala
// sempre com a mesma origem e não é preciso habilitar CORS na API.

export type Status = 'todo' | 'in_progress' | 'done'
export type Priority = 'low' | 'medium' | 'high'

export interface Task {
  id: number
  title: string
  description?: string
  priority: Priority
  status: Status
  due_date?: string
  created_at: string
  updated_at: string
}

export interface NewTask {
  title: string
  description: string
  priority: Priority
  due_date: string | null
}

// ApiError carrega o status HTTP e, quando a API informa, o campo inválido.
export class ApiError extends Error {
  readonly status: number
  readonly field?: string

  constructor(status: number, message: string, field?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.field = field
  }
}

const BASE = '/api'

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = init.body ? { 'Content-Type': 'application/json' } : undefined
  const res = await fetch(BASE + path, { ...init, headers })

  if (res.status === 204) {
    return undefined as T
  }

  const body: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    const err = (body ?? {}) as { error?: string; field?: string }
    throw new ApiError(res.status, err.error ?? `Erro HTTP ${res.status}`, err.field)
  }
  return body as T
}

export const api = {
  list: (status?: Status) => request<Task[]>(status ? `/tasks?status=${status}` : '/tasks'),
  create: (task: NewTask) => request<Task>('/tasks', { method: 'POST', body: JSON.stringify(task) }),
  changeStatus: (id: number, status: Status) =>
    request<Task>(`/tasks/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }),
  remove: (id: number) => request<void>(`/tasks/${id}`, { method: 'DELETE' }),
}
