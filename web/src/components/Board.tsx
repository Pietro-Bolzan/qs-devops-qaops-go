import { AnimatePresence, LayoutGroup, motion, type Variants } from 'motion/react'
import type { Status, Task } from '../api'
import { formatDue, relativeDue, ticket } from '../format'
import { ArrowLeftIcon, ArrowRightIcon, CalendarIcon, CheckIcon, TrashIcon } from '../icons'
import { nextStatuses, priorityLabel, statusLabel, transitionLabel } from '../labels'

const STATUSES: Status[] = ['todo', 'in_progress', 'done']

const spring = { type: 'spring', stiffness: 520, damping: 38, mass: 0.8 } as const

// direction: 1 = avançou na esteira, -1 = voltou, 0 = criada/excluída.
// O cartão sai deslizando para o lado em que a esteira andou e entra pelo lado oposto.
const cardVariants: Variants = {
  enter: (direction: number) => ({ opacity: 0, x: direction * -48, y: direction ? 0 : 12, scale: 0.97 }),
  center: { opacity: 1, x: 0, y: 0, scale: 1 },
  exit: (direction: number) => ({
    opacity: 0,
    x: direction * 48,
    scale: direction ? 1 : 0.94,
    transition: { duration: 0.18 },
  }),
}

interface BoardProps {
  tasks: Task[]
  loading: boolean
  direction: number
  onMove: (task: Task, to: Status) => void
  onRemove: (task: Task) => void
}

export function Board({ tasks, loading, direction, onMove, onRemove }: BoardProps) {
  return (
    <LayoutGroup>
      <div id="quadro" className="board" tabIndex={-1}>
        {STATUSES.map((status) => {
          const lane = tasks.filter((t) => t.status === status)
          return (
            <section key={status} className={`lane lane-${status}`} aria-labelledby={`lane-${status}`}>
              <header className="lane-head">
                <span className="lane-dot" aria-hidden="true" />
                <h2 id={`lane-${status}`}>{statusLabel[status]}</h2>
                <motion.span
                  key={lane.length}
                  className="lane-count"
                  aria-label={`${lane.length} tarefas`}
                  initial={{ scale: 0.6, opacity: 0 }}
                  animate={{ scale: 1, opacity: 1 }}
                  transition={spring}
                >
                  {lane.length}
                </motion.span>
              </header>

              <ul className="cards">
                <AnimatePresence mode="popLayout" initial={false} custom={direction}>
                  {lane.map((task) => (
                    <TaskCard key={task.id} task={task} onMove={onMove} onRemove={onRemove} />
                  ))}
                </AnimatePresence>
              </ul>

              {loading && (
                <div className="lane-empty" aria-hidden="true">
                  <span className="skeleton" />
                  <span className="skeleton short" />
                </div>
              )}
              {!loading && lane.length === 0 && <p className="lane-empty">nada aqui</p>}
            </section>
          )
        })}
      </div>
    </LayoutGroup>
  )
}

function TaskCard({ task, onMove, onRemove }: { task: Task; onMove: BoardProps['onMove']; onRemove: BoardProps['onRemove'] }) {
  const due = task.due_date ? relativeDue(task.due_date) : null

  return (
    <motion.li
      layout
      className={`card prio-${task.priority} status-${task.status}`}
      aria-label={task.title}
      variants={cardVariants}
      initial="enter"
      animate="center"
      exit="exit"
      transition={spring}
    >
      <div className="card-top">
        <span className="ticket">{ticket(task.id)}</span>
        <span className="prio">
          <span className="prio-dot" aria-hidden="true" />
          {priorityLabel[task.priority]}
        </span>
      </div>

      <h3 className="card-title">
        {task.status === 'done' && (
          <motion.span
            className="done-check"
            aria-hidden="true"
            initial={{ scale: 0, rotate: -45 }}
            animate={{ scale: 1, rotate: 0 }}
            transition={{ ...spring, delay: 0.1 }}
          >
            <CheckIcon size={12} strokeWidth={3} />
          </motion.span>
        )}
        {task.title}
      </h3>
      {task.description && <p className="card-desc">{task.description}</p>}

      <div className="card-foot">
        {task.due_date && due ? (
          <span className={`due due-${task.status === 'done' ? 'later' : due.tone}`}>
            <CalendarIcon size={13} />
            <time dateTime={task.due_date}>{formatDue(task.due_date)}</time>
            {task.status !== 'done' && <span className="due-rel">· {due.label}</span>}
          </span>
        ) : (
          <span />
        )}

        <div className="card-actions">
          {nextStatuses[task.status].map((to) => (
            <button key={to} type="button" className={`move move-${to}`} onClick={() => onMove(task, to)}>
              {to === 'todo' && <ArrowLeftIcon size={14} />}
              {transitionLabel[to]}
              {to === 'in_progress' && <ArrowRightIcon size={14} />}
              {to === 'done' && <CheckIcon size={14} />}
            </button>
          ))}
          <button type="button" className="remove" aria-label={`Excluir ${task.title}`} title="Excluir" onClick={() => onRemove(task)}>
            <TrashIcon size={15} />
          </button>
        </div>
      </div>
    </motion.li>
  )
}
