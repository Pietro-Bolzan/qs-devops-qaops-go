// Package httpapi expõe os casos de uso de tarefas como uma API REST/JSON.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"tarefas/internal/task"
)

// maxBodyBytes limita o tamanho do corpo das requisições.
const maxBodyBytes = 1 << 20

// dateLayout é o formato aceito e devolvido para o prazo (AAAA-MM-DD).
const dateLayout = time.DateOnly

type handler struct {
	svc *task.Service
	log *slog.Logger
}

// New devolve o roteador HTTP da API.
func New(svc *task.Service, logger *slog.Logger) http.Handler {
	h := &handler{svc: svc, log: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /tasks", h.list)
	mux.HandleFunc("POST /tasks", h.create)
	mux.HandleFunc("GET /tasks/{id}", h.get)
	mux.HandleFunc("PUT /tasks/{id}", h.update)
	mux.HandleFunc("PATCH /tasks/{id}/status", h.changeStatus)
	mux.HandleFunc("DELETE /tasks/{id}", h.delete)
	return mux
}

// taskRequest é o corpo aceito em POST /tasks e PUT /tasks/{id}.
type taskRequest struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Priority    task.Priority `json:"priority"`
	DueDate     *string       `json:"due_date"`
}

type statusRequest struct {
	Status task.Status `json:"status"`
}

// taskResponse é a representação JSON de uma tarefa.
type taskResponse struct {
	ID          int64         `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Priority    task.Priority `json:"priority"`
	Status      task.Status   `json:"status"`
	DueDate     *string       `json:"due_date,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type errorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List(task.Status(r.URL.Query().Get("status")))
	if err != nil {
		h.writeError(w, err)
		return
	}

	out := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, toResponse(t))
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	in, ok := h.decodeTask(w, r)
	if !ok {
		return
	}

	created, err := h.svc.Create(in)
	if err != nil {
		h.writeError(w, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/tasks/%d", created.ID))
	h.writeJSON(w, http.StatusCreated, toResponse(created))
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}

	t, err := h.svc.Get(id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, toResponse(t))
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	in, ok := h.decodeTask(w, r)
	if !ok {
		return
	}

	updated, err := h.svc.Update(id, in)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, toResponse(updated))
}

func (h *handler) changeStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req statusRequest
	if !h.decode(w, r, &req) {
		return
	}

	updated, err := h.svc.ChangeStatus(id, req.Status)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, toResponse(updated))
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pathID lê o {id} da URL; responde 400 se não for um inteiro positivo.
func (h *handler) pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id deve ser um inteiro positivo"})
		return 0, false
	}
	return id, true
}

// decodeTask lê o corpo de criação/edição e converte o prazo para time.Time.
func (h *handler) decodeTask(w http.ResponseWriter, r *http.Request) (task.Input, bool) {
	var req taskRequest
	if !h.decode(w, r, &req) {
		return task.Input{}, false
	}

	in := task.Input{Title: req.Title, Description: req.Description, Priority: req.Priority}
	if req.DueDate != nil {
		due, err := time.ParseInLocation(dateLayout, *req.DueDate, time.UTC)
		if err != nil {
			h.writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "deve estar no formato AAAA-MM-DD", Field: "due_date"})
			return task.Input{}, false
		}
		in.DueDate = &due
	}
	return in, true
}

// decode lê um JSON estrito: rejeita campos desconhecidos e corpos grandes.
func (h *handler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "JSON inválido: " + err.Error()})
		return false
	}
	return true
}

// writeError traduz erros de domínio para status HTTP.
func (h *handler) writeError(w http.ResponseWriter, err error) {
	var verr *task.ValidationError
	switch {
	case errors.As(err, &verr):
		h.writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: verr.Message, Field: verr.Field})
	case errors.Is(err, task.ErrNotFound):
		h.writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, task.ErrInvalidTransition), errors.Is(err, task.ErrTaskDone):
		h.writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	default:
		h.log.Error("erro inesperado", "erro", err)
		h.writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "erro interno"})
	}
}

func (h *handler) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.log.Error("falha ao escrever resposta", "erro", err)
	}
}

func toResponse(t task.Task) taskResponse {
	resp := taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Priority:    t.Priority,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if t.DueDate != nil {
		d := t.DueDate.Format(dateLayout)
		resp.DueDate = &d
	}
	return resp
}
