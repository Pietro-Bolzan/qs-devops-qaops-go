package task

import (
	"sync"
	"time"
)

// Clock devolve o instante atual. É injetável para que os testes usem
// uma data fixa.
type Clock func() time.Time

// Service implementa os casos de uso de tarefas sobre um Repository.
type Service struct {
	// mu serializa as operações de leitura-modificação-escrita (editar e
	// mudar status) para evitar que duas requisições simultâneas burlem
	// as regras de transição.
	mu   sync.Mutex
	repo Repository
	now  Clock
}

// NewService cria o serviço. Se clock for nil, usa o horário atual em UTC.
func NewService(repo Repository, clock Clock) *Service {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{repo: repo, now: clock}
}

// Create valida e cadastra uma nova tarefa.
func (s *Service) Create(in Input) (Task, error) {
	t, err := New(in, s.now())
	if err != nil {
		return Task{}, err
	}
	return s.repo.Create(t)
}

// Get busca uma tarefa pelo ID.
func (s *Service) Get(id int64) (Task, error) {
	return s.repo.Get(id)
}

// List devolve as tarefas, opcionalmente filtradas por status.
// Um status vazio devolve todas.
func (s *Service) List(status Status) ([]Task, error) {
	if status != "" && !status.Valid() {
		return nil, &ValidationError{Field: "status", Message: "deve ser todo, in_progress ou done"}
	}

	all, err := s.repo.List()
	if err != nil || status == "" {
		return all, err
	}

	filtered := make([]Task, 0, len(all))
	for _, t := range all {
		if t.Status == status {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

// Update edita os campos de uma tarefa existente.
func (s *Service) Update(id int64, in Input) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.repo.Get(id)
	if err != nil {
		return Task{}, err
	}
	if err := t.Apply(in, s.now()); err != nil {
		return Task{}, err
	}
	if err := s.repo.Update(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// ChangeStatus move a tarefa para outro status do ciclo de vida.
func (s *Service) ChangeStatus(id int64, to Status) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.repo.Get(id)
	if err != nil {
		return Task{}, err
	}
	if err := t.TransitionTo(to, s.now()); err != nil {
		return Task{}, err
	}
	if err := s.repo.Update(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// Delete remove uma tarefa.
func (s *Service) Delete(id int64) error {
	return s.repo.Delete(id)
}
