package task_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"tarefas/internal/task"
)

func fixedClock() time.Time { return now }

func newService() *task.Service {
	return task.NewService(task.NewMemoryStore(), fixedClock)
}

// failingRepo simula falha de persistência para verificar que o serviço
// propaga o erro em vez de engoli-lo.
type failingRepo struct {
	*task.MemoryStore
	err error
}

func (r failingRepo) Update(task.Task) error { return r.err }

func TestService_CreateAndGet(t *testing.T) {
	svc := newService()

	created, err := svc.Create(task.Input{Title: "Configurar CI", Priority: task.PriorityHigh})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.Status != task.StatusTodo || !created.CreatedAt.Equal(now) {
		t.Errorf("tarefa criada inesperada: %+v", created)
	}

	got, err := svc.Get(created.ID)
	if err != nil || got.Title != "Configurar CI" {
		t.Fatalf("Get = %+v, %v", got, err)
	}
}

func TestService_CreateInvalidDoesNotPersist(t *testing.T) {
	svc := newService()

	if _, err := svc.Create(task.Input{Title: "x"}); err == nil {
		t.Fatal("esperava erro de validação")
	}
	list, _ := svc.List("")
	if len(list) != 0 {
		t.Errorf("tarefa inválida foi persistida: %+v", list)
	}
}

func TestService_ListFilterByStatus(t *testing.T) {
	svc := newService()
	a, _ := svc.Create(task.Input{Title: "Tarefa A"})
	_, _ = svc.Create(task.Input{Title: "Tarefa B"})
	_, _ = svc.ChangeStatus(a.ID, task.StatusInProgress)

	all, _ := svc.List("")
	if len(all) != 2 {
		t.Errorf("sem filtro: len = %d, esperado 2", len(all))
	}

	inProgress, err := svc.List(task.StatusInProgress)
	if err != nil {
		t.Fatal(err)
	}
	if len(inProgress) != 1 || inProgress[0].ID != a.ID {
		t.Errorf("filtro in_progress = %+v", inProgress)
	}

	done, _ := svc.List(task.StatusDone)
	if len(done) != 0 {
		t.Errorf("filtro done deveria ser vazio: %+v", done)
	}
}

func TestService_ListInvalidStatus(t *testing.T) {
	_, err := newService().List("archived")
	assertValidationField(t, err, "status")
}

func TestService_Update(t *testing.T) {
	svc := newService()
	created, _ := svc.Create(task.Input{Title: "Antes"})

	updated, err := svc.Update(created.ID, task.Input{Title: "Depois", Priority: task.PriorityLow})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Depois" || updated.Priority != task.PriorityLow {
		t.Errorf("Update não aplicou os campos: %+v", updated)
	}

	got, _ := svc.Get(created.ID)
	if got.Title != "Depois" {
		t.Errorf("Update não persistiu: %+v", got)
	}
}

func TestService_UpdateErrors(t *testing.T) {
	svc := newService()

	if _, err := svc.Update(99, task.Input{Title: "Nada"}); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("id inexistente: esperava ErrNotFound, obteve %v", err)
	}

	created, _ := svc.Create(task.Input{Title: "Válida"})
	_, err := svc.Update(created.ID, task.Input{Title: ""})
	assertValidationField(t, err, "title")

	_, _ = svc.ChangeStatus(created.ID, task.StatusInProgress)
	_, _ = svc.ChangeStatus(created.ID, task.StatusDone)
	if _, err := svc.Update(created.ID, task.Input{Title: "Tarde demais"}); !errors.Is(err, task.ErrTaskDone) {
		t.Errorf("tarefa concluída: esperava ErrTaskDone, obteve %v", err)
	}
}

func TestService_ChangeStatus(t *testing.T) {
	svc := newService()
	created, _ := svc.Create(task.Input{Title: "Fluxo"})

	if _, err := svc.ChangeStatus(created.ID, task.StatusDone); !errors.Is(err, task.ErrInvalidTransition) {
		t.Fatalf("todo → done deveria ser rejeitada, obteve %v", err)
	}

	for _, to := range []task.Status{task.StatusInProgress, task.StatusDone} {
		got, err := svc.ChangeStatus(created.ID, to)
		if err != nil {
			t.Fatalf("transição para %s: %v", to, err)
		}
		if got.Status != to {
			t.Errorf("Status = %s, esperado %s", got.Status, to)
		}
	}

	if _, err := svc.ChangeStatus(created.ID, task.StatusTodo); !errors.Is(err, task.ErrInvalidTransition) {
		t.Errorf("done → todo deveria ser rejeitada, obteve %v", err)
	}
	if _, err := svc.ChangeStatus(404, task.StatusDone); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("id inexistente: esperava ErrNotFound, obteve %v", err)
	}
}

func TestService_Delete(t *testing.T) {
	svc := newService()
	created, _ := svc.Create(task.Input{Title: "Apagar"})

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := svc.Delete(created.ID); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("segundo Delete deveria devolver ErrNotFound, obteve %v", err)
	}
}

func TestService_PropagatesRepositoryErrors(t *testing.T) {
	boom := errors.New("falha no armazenamento")
	repo := failingRepo{MemoryStore: task.NewMemoryStore(), err: boom}
	svc := task.NewService(repo, fixedClock)
	created, _ := svc.Create(task.Input{Title: "Persistência"})

	if _, err := svc.Update(created.ID, task.Input{Title: "Nova"}); !errors.Is(err, boom) {
		t.Errorf("Update: esperava erro do repositório, obteve %v", err)
	}
	if _, err := svc.ChangeStatus(created.ID, task.StatusInProgress); !errors.Is(err, boom) {
		t.Errorf("ChangeStatus: esperava erro do repositório, obteve %v", err)
	}
}

func TestService_DefaultClockUsesUTC(t *testing.T) {
	svc := task.NewService(task.NewMemoryStore(), nil)
	created, err := svc.Create(task.Input{Title: "Relógio real"})
	if err != nil {
		t.Fatal(err)
	}
	if created.CreatedAt.Location() != time.UTC {
		t.Errorf("relógio padrão deveria usar UTC, usou %v", created.CreatedAt.Location())
	}
}

// TestService_ConcurrentStatusChange garante que, com várias requisições
// simultâneas tentando concluir a mesma tarefa, apenas uma é aceita.
func TestService_ConcurrentStatusChange(t *testing.T) {
	svc := newService()
	created, _ := svc.Create(task.Input{Title: "Disputada"})
	_, _ = svc.ChangeStatus(created.ID, task.StatusInProgress)

	const workers = 50
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		successes int
	)
	for range workers {
		wg.Go(func() {
			if _, err := svc.ChangeStatus(created.ID, task.StatusDone); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("transições aceitas = %d, esperado exatamente 1", successes)
	}
}
