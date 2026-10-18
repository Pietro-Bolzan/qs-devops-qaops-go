package task_test

import (
	"errors"
	"sync"
	"testing"

	"tarefas/internal/task"
)

func TestMemoryStore_CreateAssignsSequentialIDs(t *testing.T) {
	s := task.NewMemoryStore()
	for want := int64(1); want <= 3; want++ {
		got, err := s.Create(task.Task{Title: "Tarefa"})
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != want {
			t.Errorf("ID = %d, esperado %d", got.ID, want)
		}
	}
}

func TestMemoryStore_GetUpdateDelete(t *testing.T) {
	s := task.NewMemoryStore()
	created, _ := s.Create(task.Task{Title: "Original", Status: task.StatusTodo})

	got, err := s.Get(created.ID)
	if err != nil || got.Title != "Original" {
		t.Fatalf("Get = %+v, %v", got, err)
	}

	got.Title = "Alterada"
	if err := s.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if again, _ := s.Get(created.ID); again.Title != "Alterada" {
		t.Errorf("Update não persistiu: %q", again.Title)
	}

	if err := s.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(created.ID); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("após Delete, Get deveria devolver ErrNotFound, obteve %v", err)
	}
}

func TestMemoryStore_NotFound(t *testing.T) {
	s := task.NewMemoryStore()

	if _, err := s.Get(42); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Get: esperava ErrNotFound, obteve %v", err)
	}
	if err := s.Update(task.Task{ID: 42}); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Update: esperava ErrNotFound, obteve %v", err)
	}
	if err := s.Delete(42); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Delete: esperava ErrNotFound, obteve %v", err)
	}
}

func TestMemoryStore_ListOrderedByID(t *testing.T) {
	s := task.NewMemoryStore()
	for range 5 {
		_, _ = s.Create(task.Task{Title: "Tarefa"})
	}
	_ = s.Delete(2)

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []int64{1, 3, 4, 5}
	if len(list) != len(wantIDs) {
		t.Fatalf("len = %d, esperado %d", len(list), len(wantIDs))
	}
	for i, id := range wantIDs {
		if list[i].ID != id {
			t.Errorf("list[%d].ID = %d, esperado %d", i, list[i].ID, id)
		}
	}
}

func TestMemoryStore_ReturnsCopies(t *testing.T) {
	s := task.NewMemoryStore()
	created, _ := s.Create(task.Task{Title: "Prazo", DueDate: datePtr(2026, 12, 1)})

	// Alterar o valor devolvido não pode alterar o que está armazenado.
	*created.DueDate = created.DueDate.AddDate(1, 0, 0)
	stored, _ := s.Get(created.ID)
	if stored.DueDate.Year() != 2026 {
		t.Errorf("dado armazenado foi alterado por fora: %v", stored.DueDate)
	}
}

// TestMemoryStore_ConcurrentCreate cobre o risco R5. Deve rodar com -race.
func TestMemoryStore_ConcurrentCreate(t *testing.T) {
	const workers = 100
	s := task.NewMemoryStore()

	var wg sync.WaitGroup
	ids := make(chan int64, workers)
	for range workers {
		wg.Go(func() {
			created, err := s.Create(task.Task{Title: "Concorrente"})
			if err != nil {
				t.Error(err)
				return
			}
			ids <- created.ID
		})
	}
	wg.Wait()
	close(ids)

	seen := make(map[int64]bool, workers)
	for id := range ids {
		if seen[id] {
			t.Fatalf("ID duplicado: %d", id)
		}
		seen[id] = true
	}
	if list, _ := s.List(); len(list) != workers {
		t.Errorf("len(List) = %d, esperado %d", len(list), workers)
	}
}
