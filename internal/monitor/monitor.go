// Package monitor envia erros e panics da API para o Sentry.
//
// A integração usa só a biblioteca padrão: o pacote monta o "envelope" do
// protocolo do Sentry e o envia por HTTPS. Assim o módulo continua sem
// dependências externas e o Gate 5 (dependências) não ganha superfície nova.
//
// Sem DSN configurada, New devolve um Reporter nil, e todos os métodos de um
// Reporter nil são no-op: a API funciona igual, e os testes e o pipeline não
// dependem de rede nem de conta no Sentry.
package monitor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// queueSize limita quantos eventos esperam envio. Se o Sentry ficar lento,
// eventos excedentes são descartados em vez de acumular memória.
const queueSize = 100

// clientName identifica este cliente no cabeçalho de autenticação do Sentry.
const clientName = "tarefas-monitor/1.0"

// ErrInvalidDSN indica uma DSN fora do formato https://<chave>@<host>/<projeto>.
var ErrInvalidDSN = errors.New("SENTRY_DSN inválida")

// Config reúne a configuração do Reporter, lida de variáveis de ambiente.
type Config struct {
	DSN         string        // SENTRY_DSN; vazia desliga o monitoramento
	Environment string        // ex.: local, ci, producao
	Release     string        // versão do binário
	Timeout     time.Duration // limite de cada envio (padrão 5s)
	Client      *http.Client  // opcional; usado nos testes
	Logger      *slog.Logger  // opcional; registra falhas de envio
}

// Reporter envia eventos ao Sentry em segundo plano.
type Reporter struct {
	endpoint   string
	authHeader string
	env        string
	release    string
	serverName string
	client     *http.Client
	log        *slog.Logger

	mu     sync.Mutex // protege closed e o envio para queue
	closed bool
	queue  chan []byte
	done   chan struct{}
	failed atomic.Int64 // envios recusados ou que não chegaram ao Sentry
}

// New valida a DSN e inicia o envio em segundo plano. Com DSN vazia,
// devolve (nil, nil): o monitoramento fica desligado.
func New(cfg Config) (*Reporter, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, nil
	}
	endpoint, key, err := parseDSN(cfg.DSN)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	host, _ := os.Hostname()

	r := &Reporter{
		endpoint:   endpoint,
		authHeader: fmt.Sprintf("Sentry sentry_version=7, sentry_client=%s, sentry_key=%s", clientName, key),
		env:        cfg.Environment,
		release:    cfg.Release,
		serverName: host,
		client:     client,
		log:        logger,
		queue:      make(chan []byte, queueSize),
		done:       make(chan struct{}),
	}
	go r.worker()
	return r, nil
}

// parseDSN converte https://<chave>@<host>[/prefixo]/<projeto> no endereço
// do endpoint de envelopes e na chave pública.
func parseDSN(dsn string) (endpoint, key string, err error) {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrInvalidDSN, err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", "", fmt.Errorf("%w: esquema deve ser https", ErrInvalidDSN)
	}
	if u.User == nil || u.User.Username() == "" {
		return "", "", fmt.Errorf("%w: chave pública ausente", ErrInvalidDSN)
	}
	if u.Host == "" {
		return "", "", fmt.Errorf("%w: host ausente", ErrInvalidDSN)
	}
	path := strings.Trim(u.Path, "/")
	idx := strings.LastIndex(path, "/")
	prefix, project := "", path
	if idx >= 0 {
		prefix, project = "/"+path[:idx], path[idx+1:]
	}
	if project == "" {
		return "", "", fmt.Errorf("%w: id do projeto ausente", ErrInvalidDSN)
	}
	endpoint = fmt.Sprintf("%s://%s%s/api/%s/envelope/", u.Scheme, u.Host, prefix, project)
	return endpoint, u.User.Username(), nil
}

// Enabled informa se os eventos estão sendo enviados ao Sentry.
func (r *Reporter) Enabled() bool { return r != nil }

// Failures devolve quantos envios ao Sentry falharam (rede ou recusa).
func (r *Reporter) Failures() int64 {
	if r == nil {
		return 0
	}
	return r.failed.Load()
}

// CaptureError registra um erro inesperado (status 5xx) com os dados da
// requisição que o causou. req pode ser nil.
func (r *Reporter) CaptureError(err error, req *http.Request) {
	if r == nil || err == nil {
		return
	}
	r.enqueue(r.newEvent("error", errorType(err), err.Error(), req, 4))
}

// CapturePanic registra um panic recuperado pelo Middleware.
func (r *Reporter) CapturePanic(value any, req *http.Request) {
	if r == nil {
		return
	}
	r.enqueue(r.newEvent("fatal", "panic", fmt.Sprint(value), req, 5))
}

// CaptureMessage registra uma mensagem avulsa (usado pelo cmd/sentrycheck).
func (r *Reporter) CaptureMessage(msg string) {
	if r == nil {
		return
	}
	r.enqueue(r.newEvent("info", "mensagem", msg, nil, 4))
}

// Middleware recupera panics dos handlers, responde 500 em JSON e envia o
// panic ao Sentry. Funciona mesmo com Reporter nil (só não envia nada).
func (r *Reporter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// ErrAbortHandler é o jeito oficial de abortar uma resposta: não é erro.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}
			r.CapturePanic(rec, req)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":"erro interno"}`+"\n")
		}()
		next.ServeHTTP(w, req)
	})
}

// Close para de aceitar eventos e espera o envio dos pendentes até o
// prazo de ctx. Deve ser chamado no desligamento do servidor.
func (r *Reporter) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		close(r.queue)
	}
	r.mu.Unlock()
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Reporter) enqueue(ev event) {
	body, err := envelope(ev)
	if err != nil {
		r.log.Warn("monitor: evento não serializado", "erro", err)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		r.log.Warn("monitor: evento descartado após o encerramento")
		return
	}
	select {
	case r.queue <- body:
	default:
		r.log.Warn("monitor: fila cheia, evento descartado")
	}
}

func (r *Reporter) worker() {
	defer close(r.done)
	for body := range r.queue {
		if err := r.send(body); err != nil {
			r.failed.Add(1)
			r.log.Warn("monitor: falha ao enviar evento ao Sentry", "erro", err)
		}
	}
}

func (r *Reporter) send(body []byte) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-sentry-envelope")
	req.Header.Set("X-Sentry-Auth", r.authHeader)

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sentry respondeu %d", resp.StatusCode)
	}
	return nil
}

// event é o subconjunto do formato de evento do Sentry usado pela API.
type event struct {
	EventID     string            `json:"event_id"`
	Timestamp   string            `json:"timestamp"`
	Level       string            `json:"level"`
	Platform    string            `json:"platform"`
	Environment string            `json:"environment,omitempty"`
	Release     string            `json:"release,omitempty"`
	ServerName  string            `json:"server_name,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Request     *request          `json:"request,omitempty"`
	Exception   exceptions        `json:"exception"`
}

type exceptions struct {
	Values []exception `json:"values"`
}

type exception struct {
	Type       string     `json:"type"`
	Value      string     `json:"value"`
	Stacktrace stacktrace `json:"stacktrace"`
}

type stacktrace struct {
	Frames []frame `json:"frames"`
}

type frame struct {
	Function string `json:"function"`
	Filename string `json:"filename"`
	Lineno   int    `json:"lineno"`
	InApp    bool   `json:"in_app"`
}

type request struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

func (r *Reporter) newEvent(level, typ, value string, req *http.Request, skip int) event {
	ev := event{
		EventID:     newEventID(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		Level:       level,
		Platform:    "go",
		Environment: r.env,
		Release:     r.release,
		ServerName:  r.serverName,
		Exception: exceptions{Values: []exception{{
			Type:       typ,
			Value:      value,
			Stacktrace: stacktrace{Frames: callerFrames(skip)},
		}}},
	}
	if req != nil {
		// Só método e caminho: query string e corpo podem ter dados pessoais.
		ev.Request = &request{Method: req.Method, URL: req.URL.Path}
		ev.Tags = map[string]string{"rota": req.Method + " " + req.URL.Path}
	}
	return ev
}

// envelope serializa o evento no formato de envelope do Sentry:
// cabeçalho, cabeçalho do item e o evento, um JSON por linha.
func envelope(ev event) ([]byte, error) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	header := map[string]string{"event_id": ev.EventID, "sent_at": ev.Timestamp}
	item := map[string]any{"type": "event", "length": len(payload)}
	for _, part := range []any{header, item} {
		line, err := json.Marshal(part)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	buf.Write(payload)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// callerFrames devolve a pilha de chamadas na ordem que o Sentry espera
// (da chamada mais antiga para a mais recente).
func callerFrames(skip int) []frame {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var out []frame
	for {
		f, more := frames.Next()
		out = append(out, frame{
			Function: f.Function,
			Filename: f.File,
			Lineno:   f.Line,
			InApp:    strings.HasPrefix(f.Function, "tarefas/"),
		})
		if !more {
			break
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// errorType devolve o tipo do erro mais interno, para agrupar eventos iguais.
func errorType(err error) string {
	for {
		next := errors.Unwrap(err)
		if next == nil {
			return fmt.Sprintf("%T", err)
		}
		err = next
	}
}

func newEventID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
