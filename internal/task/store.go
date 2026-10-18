package task

import (
	"cmp"
	"slices"
	"sync"
)

// Repository abstrai a persistência das tarefas.
type Repository interface {
	Create(t Task) (Task, error)
	Get(id int64) (Task, error)
	List() ([]Task, error)
	Update(t Task) error
	Delete(id int64) error
}

// MemoryStore é um Repository em memória, seguro para uso concorrente.
type MemoryStore struct {
	mu     sync.RWMutex
	nextID int64
	tasks  map[int64]Task
}

// NewMemoryStore cria um repositório em memória vazio.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tasks: make(map[int64]Task)}
}

// Create atribui um ID sequencial à tarefa e a armazena.
func (s *MemoryStore) Create(t Task) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	t.ID = s.nextID
	s.tasks[t.ID] = clone(t)
	return clone(t), nil
}

// Get busca uma tarefa pelo ID.
func (s *MemoryStore) Get(id int64) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return clone(t), nil
}

// List devolve todas as tarefas ordenadas por ID.
func (s *MemoryStore) List() ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, clone(t))
	}
	slices.SortFunc(out, func(a, b Task) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

// Update substitui uma tarefa existente.
func (s *MemoryStore) Update(t Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	s.tasks[t.ID] = clone(t)
	return nil
}

// Delete remove uma tarefa pelo ID.
func (s *MemoryStore) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}

// clone copia o prazo para que quem chama não altere o dado armazenado
// por meio do ponteiro compartilhado.
func clone(t Task) Task {
	if t.DueDate != nil {
		d := *t.DueDate
		t.DueDate = &d
	}
	return t
}
