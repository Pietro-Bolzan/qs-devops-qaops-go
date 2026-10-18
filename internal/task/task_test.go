package task_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"tarefas/internal/task"
)

// now é o "relógio" fixo dos testes: 18/10/2026 às 15h (UTC).
var now = time.Date(2026, 10, 18, 15, 0, 0, 0, time.UTC)

func datePtr(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

// assertValidationField falha o teste se err não for um ValidationError do campo esperado.
func assertValidationField(t *testing.T, err error, field string) {
	t.Helper()
	var verr *task.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("esperava ValidationError no campo %q, obteve %v", field, err)
	}
	if verr.Field != field {
		t.Fatalf("campo inválido = %q, esperado %q", verr.Field, field)
	}
}

func TestNew_ValidInputs(t *testing.T) {
	tests := []struct {
		name         string
		in           task.Input
		wantTitle    string
		wantPriority task.Priority
	}{
		{"campos mínimos usam prioridade padrão", task.Input{Title: "Estudar Go"}, "Estudar Go", task.PriorityMedium},
		{"título nos limites mínimos", task.Input{Title: "abc"}, "abc", task.PriorityMedium},
		{"título no limite máximo", task.Input{Title: strings.Repeat("a", task.TitleMaxLen)}, strings.Repeat("a", task.TitleMaxLen), task.PriorityMedium},
		{"espaços nas pontas são removidos", task.Input{Title: "   Revisar PR   "}, "Revisar PR", task.PriorityMedium},
		{"acentos contam como um caractere", task.Input{Title: "Açã"}, "Açã", task.PriorityMedium},
		{"prioridade alta", task.Input{Title: "Deploy", Priority: task.PriorityHigh}, "Deploy", task.PriorityHigh},
		{"prioridade baixa", task.Input{Title: "Ler artigo", Priority: task.PriorityLow}, "Ler artigo", task.PriorityLow},
		{"prazo hoje é aceito", task.Input{Title: "Entregar", DueDate: datePtr(2026, 10, 18)}, "Entregar", task.PriorityMedium},
		{"prazo futuro é aceito", task.Input{Title: "Entregar", DueDate: datePtr(2026, 12, 1)}, "Entregar", task.PriorityMedium},
		{"descrição no limite máximo", task.Input{Title: "Texto", Description: strings.Repeat("d", task.DescriptionMaxLen)}, "Texto", task.PriorityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := task.New(tt.in, now)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, esperado %q", got.Title, tt.wantTitle)
			}
			if got.Priority != tt.wantPriority {
				t.Errorf("Priority = %q, esperado %q", got.Priority, tt.wantPriority)
			}
			if got.Status != task.StatusTodo {
				t.Errorf("Status = %q, toda tarefa deve nascer como todo", got.Status)
			}
			if !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
				t.Errorf("datas de criação/atualização devem ser %v", now)
			}
		})
	}
}

func TestNew_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		in    task.Input
		field string
	}{
		{"título vazio", task.Input{Title: ""}, "title"},
		{"título só com espaços", task.Input{Title: "     "}, "title"},
		{"título curto demais", task.Input{Title: "ab"}, "title"},
		{"título longo demais", task.Input{Title: strings.Repeat("a", task.TitleMaxLen+1)}, "title"},
		{"descrição longa demais", task.Input{Title: "Válido", Description: strings.Repeat("d", task.DescriptionMaxLen+1)}, "description"},
		{"prioridade desconhecida", task.Input{Title: "Válido", Priority: "urgent"}, "priority"},
		{"prazo ontem", task.Input{Title: "Válido", DueDate: datePtr(2026, 10, 17)}, "due_date"},
		{"prazo muito antigo", task.Input{Title: "Válido", DueDate: datePtr(2020, 1, 1)}, "due_date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := task.New(tt.in, now)
			assertValidationField(t, err, tt.field)
		})
	}
}

func TestValidationError_Message(t *testing.T) {
	err := &task.ValidationError{Field: "title", Message: "é obrigatório"}
	if got, want := err.Error(), "title: é obrigatório"; got != want {
		t.Errorf("Error() = %q, esperado %q", got, want)
	}
}

func TestCanTransition(t *testing.T) {
	statuses := []task.Status{task.StatusTodo, task.StatusInProgress, task.StatusDone}
	allowed := map[[2]task.Status]bool{
		{task.StatusTodo, task.StatusInProgress}: true,
		{task.StatusInProgress, task.StatusTodo}: true,
		{task.StatusInProgress, task.StatusDone}: true,
	}

	// Verifica todas as 9 combinações: só as 3 transições do ciclo de vida são permitidas.
	for _, from := range statuses {
		for _, to := range statuses {
			want := allowed[[2]task.Status{from, to}]
			if got := task.CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%s, %s) = %v, esperado %v", from, to, got, want)
			}
		}
	}
}

func TestStatusAndPriorityValid(t *testing.T) {
	for _, s := range []task.Status{task.StatusTodo, task.StatusInProgress, task.StatusDone} {
		if !s.Valid() {
			t.Errorf("status %q deveria ser válido", s)
		}
	}
	for _, s := range []task.Status{"", "TODO", "archived"} {
		if s.Valid() {
			t.Errorf("status %q deveria ser inválido", s)
		}
	}
	for _, p := range []task.Priority{"", "urgent", "HIGH"} {
		if p.Valid() {
			t.Errorf("prioridade %q deveria ser inválida", p)
		}
	}
}

func TestTransitionTo_FullLifecycle(t *testing.T) {
	tk, err := task.New(task.Input{Title: "Escrever relatório"}, now)
	if err != nil {
		t.Fatal(err)
	}

	later := now.Add(time.Hour)
	steps := []task.Status{task.StatusInProgress, task.StatusTodo, task.StatusInProgress, task.StatusDone}
	for _, to := range steps {
		if err := tk.TransitionTo(to, later); err != nil {
			t.Fatalf("transição para %s deveria ser aceita: %v", to, err)
		}
		if tk.Status != to {
			t.Fatalf("Status = %s, esperado %s", tk.Status, to)
		}
	}
	if !tk.UpdatedAt.Equal(later) {
		t.Errorf("UpdatedAt deveria ser atualizado na transição")
	}
}

func TestTransitionTo_Rejected(t *testing.T) {
	t.Run("todo direto para done", func(t *testing.T) {
		tk, _ := task.New(task.Input{Title: "Pular etapa"}, now)
		err := tk.TransitionTo(task.StatusDone, now)
		if !errors.Is(err, task.ErrInvalidTransition) {
			t.Fatalf("esperava ErrInvalidTransition, obteve %v", err)
		}
		if tk.Status != task.StatusTodo {
			t.Errorf("status não deveria mudar após transição rejeitada")
		}
	})

	t.Run("tarefa concluída não pode ser reaberta", func(t *testing.T) {
		tk, _ := task.New(task.Input{Title: "Concluída"}, now)
		_ = tk.TransitionTo(task.StatusInProgress, now)
		_ = tk.TransitionTo(task.StatusDone, now)
		for _, to := range []task.Status{task.StatusTodo, task.StatusInProgress, task.StatusDone} {
			if err := tk.TransitionTo(to, now); !errors.Is(err, task.ErrInvalidTransition) {
				t.Errorf("done → %s deveria ser rejeitada, obteve %v", to, err)
			}
		}
	})

	t.Run("status desconhecido", func(t *testing.T) {
		tk, _ := task.New(task.Input{Title: "Status estranho"}, now)
		err := tk.TransitionTo("archived", now)
		assertValidationField(t, err, "status")
	})
}

func TestApply(t *testing.T) {
	t.Run("edita campos e atualiza data", func(t *testing.T) {
		tk, _ := task.New(task.Input{Title: "Original"}, now)
		later := now.Add(2 * time.Hour)
		err := tk.Apply(task.Input{Title: " Editada ", Description: "nova", Priority: task.PriorityHigh, DueDate: datePtr(2026, 11, 1)}, later)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if tk.Title != "Editada" || tk.Description != "nova" || tk.Priority != task.PriorityHigh || tk.DueDate == nil {
			t.Errorf("campos não foram aplicados: %+v", tk)
		}
		if !tk.UpdatedAt.Equal(later) || !tk.CreatedAt.Equal(now) {
			t.Errorf("UpdatedAt deve mudar e CreatedAt deve ser mantido")
		}
	})

	t.Run("edição inválida não altera a tarefa", func(t *testing.T) {
		tk, _ := task.New(task.Input{Title: "Original"}, now)
		err := tk.Apply(task.Input{Title: ""}, now)
		assertValidationField(t, err, "title")
		if tk.Title != "Original" {
			t.Errorf("Title = %q, não deveria mudar", tk.Title)
		}
	})

	t.Run("tarefa concluída não pode ser editada", func(t *testing.T) {
		tk, _ := task.New(task.Input{Title: "Finalizada"}, now)
		_ = tk.TransitionTo(task.StatusInProgress, now)
		_ = tk.TransitionTo(task.StatusDone, now)
		if err := tk.Apply(task.Input{Title: "Nova"}, now); !errors.Is(err, task.ErrTaskDone) {
			t.Fatalf("esperava ErrTaskDone, obteve %v", err)
		}
	})
}
