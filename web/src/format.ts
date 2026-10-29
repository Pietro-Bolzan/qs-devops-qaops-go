// Datas do prazo chegam como "AAAA-MM-DD". Montamos a data com os componentes
// locais (sem passar por UTC) para não "voltar um dia" por causa do fuso.

function parseDay(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d)
}

const shortDate = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'short' })

export function formatDue(iso: string): string {
  return shortDate.format(parseDay(iso)).replace('.', '')
}

export type DueTone = 'late' | 'soon' | 'later'

// relativeDue descreve o prazo em relação a hoje: "atrasada", "vence hoje", "em 3 dias"...
export function relativeDue(iso: string, today = new Date()): { label: string; tone: DueTone } {
  const start = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  const days = Math.round((parseDay(iso).getTime() - start.getTime()) / 86_400_000)

  if (days < 0) return { label: days === -1 ? 'venceu ontem' : `atrasada ${-days} dias`, tone: 'late' }
  if (days === 0) return { label: 'vence hoje', tone: 'soon' }
  if (days === 1) return { label: 'vence amanhã', tone: 'soon' }
  if (days <= 3) return { label: `em ${days} dias`, tone: 'soon' }
  return { label: `em ${days} dias`, tone: 'later' }
}

export const ticket = (id: number) => `#${String(id).padStart(3, '0')}`
