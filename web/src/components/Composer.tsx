import { useEffect, useRef, useState, type FormEvent } from 'react'
import type { NewTask, Priority } from '../api'
import { PlusIcon } from '../icons'
import { priorityLabel } from '../labels'

const PRIORITIES: Priority[] = ['low', 'medium', 'high']

interface ComposerProps {
  invalidField?: string
  onCreate: (task: NewTask) => Promise<boolean>
}

export function Composer({ invalidField, onCreate }: ComposerProps) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<Priority>('medium')
  const [dueDate, setDueDate] = useState('')
  const [sending, setSending] = useState(false)
  const titleRef = useRef<HTMLInputElement>(null)

  // Atalho "/" leva o foco para o título (como em buscadores e no GitHub).
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      const target = event.target as HTMLElement
      const typing = target.closest('input, textarea, select, [contenteditable]')
      if (event.key === '/' && !typing) {
        event.preventDefault()
        titleRef.current?.focus()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setSending(true)
    const ok = await onCreate({ title, description, priority, due_date: dueDate || null })
    setSending(false)
    if (ok) {
      setTitle('')
      setDescription('')
      setPriority('medium')
      setDueDate('')
      titleRef.current?.focus()
    }
  }

  return (
    <form className="composer" onSubmit={handleSubmit} aria-label="Nova tarefa" noValidate>
      <div className="composer-main">
        <PlusIcon size={20} className="composer-plus" />
        <label htmlFor="title" className="visually-hidden">
          Título
        </label>
        <input
          ref={titleRef}
          id="title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="O que entra na esteira?"
          aria-invalid={invalidField === 'title'}
          aria-describedby="title-hint"
          aria-keyshortcuts="/"
          autoComplete="off"
        />
        <kbd className="kbd" aria-hidden="true">
          /
        </kbd>
      </div>

      <div className="composer-row">
        <label htmlFor="description" className="visually-hidden">
          Descrição
        </label>
        <input
          id="description"
          className="composer-desc"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Detalhes (opcional)"
          autoComplete="off"
        />

        <fieldset className="prio-picker">
          <legend className="visually-hidden">Prioridade</legend>
          {PRIORITIES.map((p) => (
            <label key={p} className={`chip chip-${p}`}>
              <input type="radio" name="priority" value={p} checked={priority === p} onChange={() => setPriority(p)} />
              <span className="prio-dot" aria-hidden="true" />
              {priorityLabel[p]}
            </label>
          ))}
        </fieldset>

        <label htmlFor="due" className="visually-hidden">
          Prazo
        </label>
        <input
          id="due"
          type="date"
          className="composer-date"
          value={dueDate}
          onChange={(e) => setDueDate(e.target.value)}
          aria-invalid={invalidField === 'due_date'}
        />

        <button type="submit" className="btn-primary" disabled={sending}>
          Adicionar
        </button>
      </div>
      <p id="title-hint" className="composer-hint">
        Título de 3 a 120 caracteres
        <span className="hint-shortcut">
          {' '}
          · pressione <kbd>/</kbd> para escrever de qualquer lugar
        </span>
      </p>
    </form>
  )
}
