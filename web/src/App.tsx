import { AnimatePresence, MotionConfig, motion } from 'motion/react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError, api, type NewTask, type Status, type Task } from './api'
import { Board } from './components/Board'
import { Composer } from './components/Composer'
import { AlertIcon, CheckIcon, LogoMark, MoonIcon, SunIcon } from './icons'
import { fieldLabel, statusLabel } from './labels'
import { useTheme } from './theme'

const REPO_URL = 'https://github.com/Pietro-Bolzan/qs-devops-qaops-go'
const ORDER: Record<Status, number> = { todo: 0, in_progress: 1, done: 2 }

interface Feedback {
  message: string
  field?: string
}

interface Toast {
  id: number
  message: string
}

function toFeedback(err: unknown): Feedback {
  if (err instanceof ApiError) {
    const label = err.field ? `${fieldLabel[err.field] ?? err.field}: ` : ''
    return { message: label + err.message, field: err.field }
  }
  return { message: 'Não foi possível falar com a API. Ela está rodando em localhost:8080?' }
}

export default function App() {
  const { theme, toggle } = useTheme()
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [online, setOnline] = useState(true)
  const [feedback, setFeedback] = useState<Feedback | null>(null)
  const [toasts, setToasts] = useState<Toast[]>([])
  const toastId = useRef(0)
  const [direction, setDirection] = useState(0)

  // Carga inicial. O "active" evita atualizar o estado depois que o
  // componente saiu da tela (ex.: no StrictMode, que monta duas vezes).
  useEffect(() => {
    let active = true
    api
      .list()
      .then((list) => active && setTasks(list))
      .catch((err: unknown) => {
        if (!active) return
        setOnline(err instanceof ApiError)
        setFeedback(toFeedback(err))
      })
      .finally(() => active && setLoading(false))
    return () => {
      active = false
    }
  }, [])

  const reload = useCallback(async () => setTasks(await api.list()), [])

  const toast = useCallback((message: string) => {
    const id = ++toastId.current
    setToasts((list) => [...list.slice(-2), { id, message }])
    setTimeout(() => setToasts((list) => list.filter((t) => t.id !== id)), 3200)
  }, [])

  // run executa uma ação na API, recarrega o quadro e avisa o resultado.
  async function run(action: () => Promise<unknown>, success: string): Promise<boolean> {
    try {
      await action()
      setOnline(true)
      setFeedback(null)
      toast(success)
      await reload()
      return true
    } catch (err) {
      setOnline(err instanceof ApiError)
      setFeedback(toFeedback(err))
      return false
    }
  }

  const create = (task: NewTask) => {
    setDirection(0)
    return run(() => api.create(task), `Tarefa "${task.title.trim()}" entrou na esteira.`)
  }
  const move = (task: Task, to: Status) => {
    setDirection(ORDER[to] > ORDER[task.status] ? 1 : -1)
    return run(() => api.changeStatus(task.id, to), `"${task.title}" foi para ${statusLabel[to].toLowerCase()}.`)
  }
  const remove = (task: Task) => {
    setDirection(0)
    return run(() => api.remove(task.id), `"${task.title}" excluída.`)
  }

  const done = tasks.filter((t) => t.status === 'done').length
  const progress = tasks.length ? Math.round((done / tasks.length) * 100) : 0

  return (
    <MotionConfig reducedMotion="user">
      <a href="#quadro" className="skip-link">
        Pular para o quadro
      </a>

      <header className="topbar">
        <div className="topbar-inner">
          <span className="brand">
            <LogoMark />
            <span className="brand-name">esteira</span>
          </span>
          <div className="topbar-actions">
            <span className={`api-pill ${online ? 'is-online' : 'is-offline'}`}>
              <span className="api-dot" aria-hidden="true" />
              API {online ? 'conectada' : 'sem conexão'}
            </span>
            <button
              type="button"
              className="icon-btn"
              onClick={toggle}
              aria-label={theme === 'dark' ? 'Ativar modo claro' : 'Ativar modo escuro'}
              title={theme === 'dark' ? 'Modo claro' : 'Modo escuro'}
            >
              <AnimatePresence mode="wait" initial={false}>
                <motion.span
                  key={theme}
                  className="icon-swap"
                  initial={{ rotate: -90, opacity: 0, scale: 0.6 }}
                  animate={{ rotate: 0, opacity: 1, scale: 1 }}
                  exit={{ rotate: 90, opacity: 0, scale: 0.6 }}
                  transition={{ duration: 0.18 }}
                >
                  {theme === 'dark' ? <SunIcon size={18} /> : <MoonIcon size={18} />}
                </motion.span>
              </AnimatePresence>
            </button>
          </div>
        </div>
      </header>

      <main className="page">
        <motion.section
          className="hero"
          aria-labelledby="titulo"
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
        >
          <p className="eyebrow">Qualidade de Software · UFSM</p>
          <h1 id="titulo">
            Do backlog ao <span className="accent-text">feito</span>.
          </h1>
          <p className="lede">
            Cada tarefa passa por três estações. A API decide o que pode avançar, e o pipeline de CI garante que essas
            regras não quebram.
          </p>

          <div className="progress">
            <div className="progress-text">
              <span>
                <strong>{done}</strong> de {tasks.length} concluídas
              </span>
              <span className="progress-pct">{progress}%</span>
            </div>
            <progress className="visually-hidden" value={progress} max={100} aria-label="Tarefas concluídas">
              {progress}%
            </progress>
            <div className="progress-track" aria-hidden="true">
              <motion.div
                className="progress-bar"
                initial={false}
                animate={{ width: `${progress}%` }}
                transition={{ type: 'spring', stiffness: 120, damping: 20 }}
              />
            </div>
          </div>
        </motion.section>

        <Composer invalidField={feedback?.field} onCreate={create} />

        <div role="alert" className="alert-slot">
          <AnimatePresence>
            {feedback && (
              <motion.div
                key={feedback.message}
                className="alert"
                initial={{ opacity: 0, y: -8, height: 0 }}
                animate={{ opacity: 1, y: 0, height: 'auto' }}
                exit={{ opacity: 0, y: -8, height: 0 }}
              >
                <AlertIcon size={18} />
                <span>{feedback.message}</span>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        <Board tasks={tasks} loading={loading} direction={direction} onMove={move} onRemove={remove} />
      </main>

      <footer className="footer">
        <span>Trabalho prático de Qualidade de Software · Universidade Federal de Santa Maria</span>
        <a href={REPO_URL}>Código-fonte</a>
      </footer>

      <output className="toasts" aria-live="polite">
        <AnimatePresence>
          {toasts.map((t) => (
            <motion.div
              key={t.id}
              layout
              className="toast"
              initial={{ opacity: 0, y: 24, scale: 0.96 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, x: 40, transition: { duration: 0.2 } }}
              transition={{ type: 'spring', stiffness: 420, damping: 32 }}
            >
              <span className="toast-icon" aria-hidden="true">
                <CheckIcon size={14} strokeWidth={3} />
              </span>
              {t.message}
            </motion.div>
          ))}
        </AnimatePresence>
      </output>
    </MotionConfig>
  )
}
