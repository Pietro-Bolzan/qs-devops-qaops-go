import type { Priority, Status } from './api'

export const statusLabel: Record<Status, string> = {
  todo: 'A fazer',
  in_progress: 'Em andamento',
  done: 'Concluída',
}

export const priorityLabel: Record<Priority, string> = {
  low: 'Baixa',
  medium: 'Média',
  high: 'Alta',
}

export const fieldLabel: Record<string, string> = {
  title: 'Título',
  description: 'Descrição',
  priority: 'Prioridade',
  due_date: 'Prazo',
  status: 'Status',
}

// Espelha o ciclo de vida da API (RN06 e RN07) só para decidir quais botões
// mostrar. Quem garante a regra é a API: se o front errar, ela responde 409.
export const nextStatuses: Record<Status, Status[]> = {
  todo: ['in_progress'],
  in_progress: ['todo', 'done'],
  done: [],
}

export const transitionLabel: Record<Status, string> = {
  todo: 'Voltar',
  in_progress: 'Iniciar',
  done: 'Concluir',
}
