package httpapi_test

// Testes de API/integração: sobem a API completa (handler + serviço +
// repositório em memória) em um servidor httptest e verificam o contrato
// HTTP (status, cabeçalhos e JSON) de cada endpoint.

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tarefas/internal/httpapi"
	"tarefas/internal/task"
)

var now = time.Date(2026, 10, 18, 15, 0, 0, 0, time.UTC)

type apiTask struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	DueDate     string `json:"due_date"`
}

type apiError struct {
	Error string `json:"error"`
	Field string `json:"field"`
}

func newServer(t *testing.T, repo task.Repository) *httptest.Server {
	t.Helper()
	if repo == nil {
		repo = task.NewMemoryStore()
	}
	svc := task.NewService(repo, func() time.Time { return now })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(httpapi.New(svc, logger))
	t.Cleanup(srv.Close)
	return srv
}

// call faz a requisição e decodifica o JSON da resposta em out (se não for nil).
func call(t *testing.T, srv *httptest.Server, method, path, body string, out any) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("resposta não é JSON válido (%s): %q", err, raw)
		}
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("%s %s: status = %d, esperado %d", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, want)
	}
}

func createTask(t *testing.T, srv *httptest.Server, body string) apiTask {
	t.Helper()
	var created apiTask
	assertStatus(t, call(t, srv, http.MethodPost, "/tasks", body, &created), http.StatusCreated)
	return created
}

func TestHealth(t *testing.T) {
	srv := newServer(t, nil)
	var body map[string]string
	resp := call(t, srv, http.MethodGet, "/health", "", &body)

	assertStatus(t, resp, http.StatusOK)
	if body["status"] != "ok" {
		t.Errorf("body = %v", body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestCreateTask(t *testing.T) {
	srv := newServer(t, nil)

	var created apiTask
	resp := call(t, srv, http.MethodPost, "/tasks",
		`{"title":"Configurar pipeline","priority":"high","due_date":"2026-10-30"}`, &created)

	assertStatus(t, resp, http.StatusCreated)
	if loc := resp.Header.Get("Location"); loc != "/tasks/1" {
		t.Errorf("Location = %q, esperado /tasks/1", loc)
	}
	want := apiTask{ID: 1, Title: "Configurar pipeline", Priority: "high", Status: "todo", DueDate: "2026-10-30"}
	if created != want {
		t.Errorf("tarefa criada = %+v, esperado %+v", created, want)
	}
}

func TestCreateTask_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		field string
	}{
		{"título ausente", `{"description":"sem título"}`, "title"},
		{"título curto", `{"title":"ab"}`, "title"},
		{"prioridade inválida", `{"title":"Válido","priority":"urgent"}`, "priority"},
		{"prazo no passado", `{"title":"Válido","due_date":"2026-10-17"}`, "due_date"},
		{"prazo em formato errado", `{"title":"Válido","due_date":"30/10/2026"}`, "due_date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newServer(t, nil)
			var body apiError
			resp := call(t, srv, http.MethodPost, "/tasks", tt.body, &body)

			assertStatus(t, resp, http.StatusUnprocessableEntity)
			if body.Field != tt.field || body.Error == "" {
				t.Errorf("erro = %+v, esperado campo %q", body, tt.field)
			}

			// Nada pode ter sido persistido.
			var list []apiTask
			call(t, srv, http.MethodGet, "/tasks", "", &list)
			if len(list) != 0 {
				t.Errorf("tarefa inválida foi persistida: %+v", list)
			}
		})
	}
}

func TestCreateTask_MalformedBody(t *testing.T) {
	tests := map[string]string{
		"JSON quebrado":       `{"title":`,
		"campo desconhecido":  `{"title":"Válido","owner":"alguém"}`,
		"tipo errado":         `{"title":123}`,
		"corpo vazio":         ``,
		"corpo grande demais": `{"title":"` + strings.Repeat("a", 2<<20) + `"}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			srv := newServer(t, nil)
			var errBody apiError
			resp := call(t, srv, http.MethodPost, "/tasks", body, &errBody)
			assertStatus(t, resp, http.StatusBadRequest)
			if errBody.Error == "" {
				t.Error("resposta de erro deveria ter mensagem")
			}
		})
	}
}

func TestGetTask(t *testing.T) {
	srv := newServer(t, nil)
	created := createTask(t, srv, `{"title":"Buscar","description":"detalhes"}`)

	var got apiTask
	assertStatus(t, call(t, srv, http.MethodGet, "/tasks/1", "", &got), http.StatusOK)
	if got != created {
		t.Errorf("GET = %+v, esperado %+v", got, created)
	}

	var notFound apiError
	assertStatus(t, call(t, srv, http.MethodGet, "/tasks/999", "", &notFound), http.StatusNotFound)

	for _, bad := range []string{"abc", "0", "-1"} {
		assertStatus(t, call(t, srv, http.MethodGet, "/tasks/"+bad, "", &apiError{}), http.StatusBadRequest)
	}
}

func TestListTasks(t *testing.T) {
	srv := newServer(t, nil)

	var empty []apiTask
	assertStatus(t, call(t, srv, http.MethodGet, "/tasks", "", &empty), http.StatusOK)
	if empty == nil || len(empty) != 0 {
		t.Errorf("lista vazia deve ser [] e não null: %v", empty)
	}

	createTask(t, srv, `{"title":"Primeira"}`)
	createTask(t, srv, `{"title":"Segunda"}`)
	assertStatus(t, call(t, srv, http.MethodPatch, "/tasks/2/status", `{"status":"in_progress"}`, nil), http.StatusOK)

	var all, inProgress []apiTask
	call(t, srv, http.MethodGet, "/tasks", "", &all)
	call(t, srv, http.MethodGet, "/tasks?status=in_progress", "", &inProgress)
	if len(all) != 2 {
		t.Errorf("len(all) = %d, esperado 2", len(all))
	}
	if len(inProgress) != 1 || inProgress[0].ID != 2 {
		t.Errorf("filtro in_progress = %+v", inProgress)
	}

	var bad apiError
	assertStatus(t, call(t, srv, http.MethodGet, "/tasks?status=archived", "", &bad), http.StatusUnprocessableEntity)
	if bad.Field != "status" {
		t.Errorf("campo = %q, esperado status", bad.Field)
	}
}

func TestUpdateTask(t *testing.T) {
	srv := newServer(t, nil)
	createTask(t, srv, `{"title":"Antes"}`)

	var updated apiTask
	resp := call(t, srv, http.MethodPut, "/tasks/1", `{"title":"Depois","priority":"low","due_date":"2026-11-01"}`, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Title != "Depois" || updated.Priority != "low" || updated.DueDate != "2026-11-01" {
		t.Errorf("tarefa editada = %+v", updated)
	}

	assertStatus(t, call(t, srv, http.MethodPut, "/tasks/1", `{"title":""}`, &apiError{}), http.StatusUnprocessableEntity)
	assertStatus(t, call(t, srv, http.MethodPut, "/tasks/1", `{"title":`, &apiError{}), http.StatusBadRequest)
	assertStatus(t, call(t, srv, http.MethodPut, "/tasks/42", `{"title":"Nada"}`, &apiError{}), http.StatusNotFound)
	assertStatus(t, call(t, srv, http.MethodPut, "/tasks/x", `{"title":"Nada"}`, &apiError{}), http.StatusBadRequest)
}

func TestChangeStatus(t *testing.T) {
	srv := newServer(t, nil)
	createTask(t, srv, `{"title":"Ciclo de vida"}`)

	var conflict apiError
	assertStatus(t, call(t, srv, http.MethodPatch, "/tasks/1/status", `{"status":"done"}`, &conflict), http.StatusConflict)
	if !strings.Contains(conflict.Error, "transição") {
		t.Errorf("mensagem de conflito = %q", conflict.Error)
	}

	assertStatus(t, call(t, srv, http.MethodPatch, "/tasks/1/status", `{"status":"archived"}`, &apiError{}), http.StatusUnprocessableEntity)
	assertStatus(t, call(t, srv, http.MethodPatch, "/tasks/1/status", `{"state":"done"}`, &apiError{}), http.StatusBadRequest)
	assertStatus(t, call(t, srv, http.MethodPatch, "/tasks/9/status", `{"status":"in_progress"}`, &apiError{}), http.StatusNotFound)
	assertStatus(t, call(t, srv, http.MethodPatch, "/tasks/x/status", `{"status":"in_progress"}`, &apiError{}), http.StatusBadRequest)
}

func TestDeleteTask(t *testing.T) {
	srv := newServer(t, nil)
	createTask(t, srv, `{"title":"Apagar"}`)

	assertStatus(t, call(t, srv, http.MethodDelete, "/tasks/1", "", nil), http.StatusNoContent)
	assertStatus(t, call(t, srv, http.MethodGet, "/tasks/1", "", &apiError{}), http.StatusNotFound)
	assertStatus(t, call(t, srv, http.MethodDelete, "/tasks/1", "", &apiError{}), http.StatusNotFound)
	assertStatus(t, call(t, srv, http.MethodDelete, "/tasks/abc", "", &apiError{}), http.StatusBadRequest)
}

func TestMethodNotAllowed(t *testing.T) {
	srv := newServer(t, nil)
	resp := call(t, srv, http.MethodPatch, "/tasks", "", nil)
	assertStatus(t, resp, http.StatusMethodNotAllowed)
}

// TestTaskLifecycle é o cenário de integração principal: percorre todo o
// ciclo de vida de uma tarefa pela API, como um cliente real faria.
func TestTaskLifecycle(t *testing.T) {
	srv := newServer(t, nil)
	created := createTask(t, srv, `{"title":"Entregar trabalho de QS","priority":"high","due_date":"2026-11-20"}`)

	for _, status := range []string{"in_progress", "done"} {
		var got apiTask
		assertStatus(t, call(t, srv, http.MethodPatch, "/tasks/1/status", `{"status":"`+status+`"}`, &got), http.StatusOK)
		if got.Status != status {
			t.Fatalf("status = %q, esperado %q", got.Status, status)
		}
	}

	// Tarefa concluída é final: não pode ser reaberta nem editada.
	assertStatus(t, call(t, srv, http.MethodPatch, "/tasks/1/status", `{"status":"todo"}`, &apiError{}), http.StatusConflict)
	assertStatus(t, call(t, srv, http.MethodPut, "/tasks/1", `{"title":"Reescrever"}`, &apiError{}), http.StatusConflict)

	var final apiTask
	call(t, srv, http.MethodGet, "/tasks/1", "", &final)
	if final.Status != "done" || final.Title != created.Title {
		t.Errorf("estado final inesperado: %+v", final)
	}
}

// brokenRepo simula falha de infraestrutura no armazenamento.
type brokenRepo struct{ *task.MemoryStore }

func (brokenRepo) Update(task.Task) error { return errors.New("disco cheio") }

func TestInternalErrorIsNotLeaked(t *testing.T) {
	srv := newServer(t, brokenRepo{task.NewMemoryStore()})
	createTask(t, srv, `{"title":"Falha"}`)

	var body apiError
	resp := call(t, srv, http.MethodPatch, "/tasks/1/status", `{"status":"in_progress"}`, &body)
	assertStatus(t, resp, http.StatusInternalServerError)
	if body.Error != "erro interno" {
		t.Errorf("detalhe interno vazou para o cliente: %q", body.Error)
	}
}

// TestResponseIsValidJSON garante que todas as respostas de erro também são JSON.
func TestResponseIsValidJSON(t *testing.T) {
	srv := newServer(t, nil)
	resp := call(t, srv, http.MethodGet, "/tasks/abc", "", nil)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type de erro = %q", ct)
	}
}
