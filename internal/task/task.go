// Package task contém o domínio do gerenciador de tarefas: a entidade
// Tarefa, suas regras de validação e o ciclo de vida de status.
package task

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Limites das regras de negócio (RN01 e RN02 em docs/sistema.md).
const (
	TitleMinLen       = 3
	TitleMaxLen       = 120
	DescriptionMaxLen = 1000
)

// Status representa a etapa do ciclo de vida de uma tarefa.
type Status string

// Status possíveis de uma tarefa.
const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// Priority representa a prioridade de uma tarefa.
type Priority string

// Prioridades aceitas.
const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Erros de domínio. A camada HTTP usa errors.Is para traduzi-los em status.
var (
	ErrNotFound          = errors.New("tarefa não encontrada")
	ErrInvalidTransition = errors.New("transição de status não permitida")
	ErrTaskDone          = errors.New("tarefa concluída não pode ser alterada")
)

// ValidationError indica que um campo não atende a uma regra de negócio.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// allowedTransitions define o ciclo de vida (RN06 e RN07):
// todo → in_progress → done, podendo devolver in_progress para todo.
// done é um estado final.
var allowedTransitions = map[Status][]Status{
	StatusTodo:       {StatusInProgress},
	StatusInProgress: {StatusTodo, StatusDone},
	StatusDone:       {},
}

// Valid informa se o status é um dos valores conhecidos.
func (s Status) Valid() bool {
	_, ok := allowedTransitions[s]
	return ok
}

// Valid informa se a prioridade é um dos valores conhecidos.
func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	default:
		return false
	}
}

// CanTransition informa se a mudança de status from → to é permitida.
func CanTransition(from, to Status) bool {
	return slices.Contains(allowedTransitions[from], to)
}

// Task é a entidade principal do domínio.
type Task struct {
	ID          int64
	Title       string
	Description string
	Priority    Priority
	Status      Status
	DueDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Input reúne os campos editáveis de uma tarefa, usados na criação e na edição.
type Input struct {
	Title       string
	Description string
	Priority    Priority
	DueDate     *time.Time
}

// normalize remove espaços nas pontas e aplica a prioridade padrão (RN03).
func (in Input) normalize() Input {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.Priority == "" {
		in.Priority = PriorityMedium
	}
	return in
}

// Validate verifica as regras RN01 a RN04 e devolve o primeiro erro encontrado.
// O Input deve estar normalizado.
func (in Input) Validate(now time.Time) error {
	titleLen := utf8.RuneCountInString(in.Title)
	switch {
	case titleLen == 0:
		return &ValidationError{Field: "title", Message: "é obrigatório"}
	case titleLen < TitleMinLen:
		return &ValidationError{Field: "title", Message: fmt.Sprintf("deve ter ao menos %d caracteres", TitleMinLen)}
	case titleLen > TitleMaxLen:
		return &ValidationError{Field: "title", Message: fmt.Sprintf("deve ter no máximo %d caracteres", TitleMaxLen)}
	}

	if utf8.RuneCountInString(in.Description) > DescriptionMaxLen {
		return &ValidationError{Field: "description", Message: fmt.Sprintf("deve ter no máximo %d caracteres", DescriptionMaxLen)}
	}

	if !in.Priority.Valid() {
		return &ValidationError{Field: "priority", Message: "deve ser low, medium ou high"}
	}

	if in.DueDate != nil && in.DueDate.Before(startOfDay(now)) {
		return &ValidationError{Field: "due_date", Message: "não pode ser anterior à data atual"}
	}

	return nil
}

// New cria uma tarefa válida com status inicial todo (RN05).
func New(in Input, now time.Time) (Task, error) {
	in = in.normalize()
	if err := in.Validate(now); err != nil {
		return Task{}, err
	}

	return Task{
		Title:       in.Title,
		Description: in.Description,
		Priority:    in.Priority,
		Status:      StatusTodo,
		DueDate:     in.DueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Apply substitui os campos editáveis da tarefa. Tarefas concluídas não
// podem ser editadas (RN07).
func (t *Task) Apply(in Input, now time.Time) error {
	if t.Status == StatusDone {
		return ErrTaskDone
	}

	in = in.normalize()
	if err := in.Validate(now); err != nil {
		return err
	}

	t.Title = in.Title
	t.Description = in.Description
	t.Priority = in.Priority
	t.DueDate = in.DueDate
	t.UpdatedAt = now
	return nil
}

// TransitionTo muda o status da tarefa respeitando o ciclo de vida (RN06 e RN07).
func (t *Task) TransitionTo(to Status, now time.Time) error {
	if !to.Valid() {
		return &ValidationError{Field: "status", Message: "deve ser todo, in_progress ou done"}
	}
	if !CanTransition(t.Status, to) {
		return fmt.Errorf("%w: %s → %s", ErrInvalidTransition, t.Status, to)
	}

	t.Status = to
	t.UpdatedAt = now
	return nil
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
