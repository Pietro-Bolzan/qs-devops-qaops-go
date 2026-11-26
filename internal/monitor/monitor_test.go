package monitor_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"tarefas/internal/monitor"
)

// fakeSentry simula o endpoint de envelopes do Sentry e guarda o que recebeu.
type fakeSentry struct {
	mu       sync.Mutex
	paths    []string
	auths    []string
	events   []sentEvent
	status   int
	received chan struct{}
}

func newFakeSentry(t *testing.T, status int) (*fakeSentry, *httptest.Server) {
	t.Helper()
	f := &fakeSentry{status: status, received: make(chan struct{}, 10)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("lendo envelope: %v", err)
		}
		f.mu.Lock()
		f.paths = append(f.paths, r.URL.Path)
		f.auths = append(f.auths, r.Header.Get("X-Sentry-Auth"))
		f.events = append(f.events, parseEnvelope(t, body))
		f.mu.Unlock()
		w.WriteHeader(f.status)
		f.received <- struct{}{}
	}))
	t.Cleanup(srv.Close)
	return f, srv
}

// sentEvent é o evento como o Sentry o recebe (só os campos verificados).
type sentEvent struct {
	Level       string            `json:"level"`
	Environment string            `json:"environment"`
	Release     string            `json:"release"`
	Tags        map[string]string `json:"tags"`
	Request     *struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"request"`
	Exception struct {
		Values []sentException `json:"values"`
	} `json:"exception"`
}

type sentException struct {
	Type       string `json:"type"`
	Value      string `json:"value"`
	Stacktrace struct {
		Frames []struct {
			Function string `json:"function"`
		} `json:"frames"`
	} `json:"stacktrace"`
}

// parseEnvelope confere as três linhas do envelope e devolve o evento.
func parseEnvelope(t *testing.T, body []byte) sentEvent {
	t.Helper()
	sc := bufio.NewScanner(bytes.NewReader(body))
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if len(lines) != 3 {
		t.Fatalf("envelope deveria ter 3 linhas, tem %d: %q", len(lines), body)
	}
	var item struct {
		Type   string `json:"type"`
		Length int    `json:"length"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &item); err != nil {
		t.Fatal(err)
	}
	if item.Type != "event" || item.Length != len(lines[2]) {
		t.Errorf("cabeçalho do item inválido: %+v (evento com %d bytes)", item, len(lines[2]))
	}
	var ev sentEvent
	if err := json.Unmarshal([]byte(lines[2]), &ev); err != nil {
		t.Fatal(err)
	}
	return ev
}

func (f *fakeSentry) wait(t *testing.T) {
	t.Helper()
	select {
	case <-f.received:
	case <-time.After(3 * time.Second):
		t.Fatal("o Sentry falso não recebeu o evento")
	}
}

func (f *fakeSentry) last() sentEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.events[len(f.events)-1]
}

func dsnFor(srv *httptest.Server) string {
	return strings.Replace(srv.URL, "http://", "http://chave-publica@", 1) + "/42"
}

func newReporter(t *testing.T, dsn string) *monitor.Reporter {
	t.Helper()
	rep, err := monitor.New(monitor.Config{DSN: dsn, Environment: "teste", Release: "v1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rep.Close(context.Background()) })
	return rep
}

func exceptionOf(t *testing.T, ev sentEvent) sentException {
	t.Helper()
	if len(ev.Exception.Values) != 1 {
		t.Fatalf("evento deveria ter 1 exceção, tem %d", len(ev.Exception.Values))
	}
	return ev.Exception.Values[0]
}

func TestNewSemDSNDesligaMonitoramento(t *testing.T) {
	rep, err := monitor.New(monitor.Config{DSN: "  "})
	if err != nil || rep != nil {
		t.Fatalf("New sem DSN = (%v, %v), quer (nil, nil)", rep, err)
	}
	if rep.Enabled() {
		t.Error("Reporter nil não deveria estar habilitado")
	}
	// Todos os métodos de um Reporter nil são no-op.
	rep.CaptureError(errors.New("x"), nil)
	rep.CapturePanic("x", nil)
	rep.CaptureMessage("x")
	if err := rep.Close(context.Background()); err != nil {
		t.Errorf("Close em Reporter nil = %v", err)
	}
}

func TestNewRejeitaDSNInvalida(t *testing.T) {
	casos := map[string]string{
		"esquema errado": "ftp://chave@sentry.io/1",
		"sem chave":      "https://sentry.io/1",
		"sem host":       "https://chave@/1",
		"sem projeto":    "https://chave@sentry.io/",
		"url malformada": "https://chave@sentry.io/%zz",
		"só texto solto": "isto-nao-e-uma-dsn",
	}
	for nome, dsn := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := monitor.New(monitor.Config{DSN: dsn})
			if !errors.Is(err, monitor.ErrInvalidDSN) {
				t.Errorf("New(%q) erro = %v, quer ErrInvalidDSN", dsn, err)
			}
		})
	}
}

func TestCaptureErrorEnviaEnvelopeAoSentry(t *testing.T) {
	fake, srv := newFakeSentry(t, http.StatusOK)
	rep := newReporter(t, dsnFor(srv))
	if !rep.Enabled() {
		t.Fatal("Reporter com DSN deveria estar habilitado")
	}

	req := httptest.NewRequest(http.MethodPost, "/tasks?email=pessoa@exemplo.com", nil)
	rep.CaptureError(fmt.Errorf("salvando: %w", io.ErrUnexpectedEOF), req)
	fake.wait(t)

	if got := fake.paths[0]; got != "/api/42/envelope/" {
		t.Errorf("endpoint = %q, quer /api/42/envelope/", got)
	}
	if !strings.Contains(fake.auths[0], "sentry_key=chave-publica") {
		t.Errorf("cabeçalho de autenticação sem a chave: %q", fake.auths[0])
	}

	ev := fake.last()
	if ev.Level != "error" || ev.Environment != "teste" || ev.Release != "v1.2.3" {
		t.Errorf("metadados do evento incorretos: %+v", ev)
	}
	exc := exceptionOf(t, ev)
	if exc.Type != "*errors.errorString" || exc.Value != "salvando: unexpected EOF" {
		t.Errorf("exceção = %+v", exc)
	}
	if ev.Request == nil || ev.Request.Method != http.MethodPost || ev.Request.URL != "/tasks" {
		t.Errorf("a requisição deve ir sem a query string (dados pessoais): %+v", ev.Request)
	}
	if len(exc.Stacktrace.Frames) == 0 {
		t.Error("evento sem pilha de chamadas")
	}
}

func TestCaptureMessageEFalhaDoSentryNaoDerrubamAAPI(t *testing.T) {
	fake, srv := newFakeSentry(t, http.StatusTooManyRequests)
	rep := newReporter(t, dsnFor(srv))

	rep.CaptureMessage("teste de integração")
	rep.CaptureError(nil, nil) // erro nil é ignorado
	fake.wait(t)

	ev := fake.last()
	if ev.Level != "info" || exceptionOf(t, ev).Value != "teste de integração" {
		t.Errorf("evento de mensagem incorreto: %+v", ev)
	}
	if ev.Request != nil {
		t.Error("evento sem requisição não deveria ter o campo request")
	}
}

func TestEnvioComSentryForaDoArNaoTrava(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	dsn := dsnFor(srv)
	srv.Close() // ninguém escutando: o envio falha

	rep, err := monitor.New(monitor.Config{DSN: dsn, Timeout: 200 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	rep.CaptureMessage("vai falhar")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rep.Close(ctx); err != nil {
		t.Fatalf("Close deveria terminar mesmo com o Sentry fora do ar: %v", err)
	}
	rep.CaptureMessage("depois do Close é descartado")
	if err := rep.Close(ctx); err != nil {
		t.Errorf("Close repetido = %v", err)
	}
}

func TestCloseRespeitaPrazo(t *testing.T) {
	bloqueio := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-bloqueio }))
	t.Cleanup(func() { close(bloqueio); srv.Close() })

	rep, err := monitor.New(monitor.Config{DSN: dsnFor(srv), Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	rep.CaptureMessage("lento")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := rep.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Close = %v, quer DeadlineExceeded", err)
	}
}

func TestFilaCheiaDescartaEventos(t *testing.T) {
	bloqueio := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-bloqueio }))
	t.Cleanup(func() { close(bloqueio); srv.Close() })

	rep, err := monitor.New(monitor.Config{DSN: dsnFor(srv)})
	if err != nil {
		t.Fatal(err)
	}
	// Mais eventos que a fila comporta: os excedentes são descartados sem bloquear.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 500; i++ {
			rep.CaptureMessage("evento")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("CaptureMessage bloqueou com a fila cheia")
	}
}

func TestMiddlewareRecuperaPanicEEnviaAoSentry(t *testing.T) {
	fake, srv := newFakeSentry(t, http.StatusOK)
	rep := newReporter(t, dsnFor(srv))

	h := rep.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("índice fora do intervalo")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/7", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, quer 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "erro interno") {
		t.Errorf("corpo = %q", rec.Body.String())
	}
	fake.wait(t)
	ev := fake.last()
	if ev.Level != "fatal" || exceptionOf(t, ev).Value != "índice fora do intervalo" {
		t.Errorf("evento de panic incorreto: %+v", ev)
	}
	if ev.Tags["rota"] != "GET /tasks/7" {
		t.Errorf("tag rota = %q", ev.Tags["rota"])
	}
}

func TestMiddlewareSemSentryAindaRecuperaPanic(t *testing.T) {
	var rep *monitor.Reporter
	h := rep.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, quer 500", rec.Code)
	}
}

func TestMiddlewareRepassaErrAbortHandler(t *testing.T) {
	var rep *monitor.Reporter
	h := rep.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler { //nolint:errorlint // comparação do valor do panic
			t.Errorf("panic repassado = %v, quer http.ErrAbortHandler", rec)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestMiddlewareSemPanicNaoInterfere(t *testing.T) {
	var rep *monitor.Reporter
	h := rep.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, quer 418", rec.Code)
	}
}
