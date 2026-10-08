package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/abhaypandey/ledgerqueue/internal/job"
	"github.com/abhaypandey/ledgerqueue/internal/metrics"
	"github.com/abhaypandey/ledgerqueue/internal/payment"
	"github.com/abhaypandey/ledgerqueue/internal/queue"
	"github.com/abhaypandey/ledgerqueue/internal/store"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	Store         store.Store
	Queue         queue.Queue
	WebhookSecret string
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /ready", a.ready)
	mux.HandleFunc("GET /metrics", promhttp.Handler().ServeHTTP)
	mux.HandleFunc("POST /jobs", a.createJob)
	mux.HandleFunc("GET /jobs/{id}", a.getJob)
	mux.HandleFunc("POST /webhooks/payment", a.paymentWebhook)
	mux.HandleFunc("GET /reconciliation/latest", a.latestReconciliation)
	return mux
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "store unavailable")
		return
	}
	if err := a.Queue.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "queue unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) createJob(w http.ResponseWriter, r *http.Request) {
	var req job.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if err := a.enqueueNewJob(w, r, req); err != nil {
		return
	}
}

func (a *API) paymentWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body")
		return
	}
	sig := r.Header.Get("X-Webhook-Signature")
	if !payment.Verify(body, a.WebhookSecret, sig) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	var ev payment.WebhookEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !ev.Valid() {
		writeError(w, http.StatusBadRequest, "invalid event")
		return
	}

	if existing, err := a.Store.GetByIdempotencyKey(r.Context(), ev.EventID); err == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "duplicate",
			"job_id": existing.ID,
		})
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}

	payload, _ := json.Marshal(ev)
	req := job.CreateRequest{
		Type:           job.TypeProcessWebhook,
		Payload:        payload,
		IdempotencyKey: ev.EventID,
	}
	j := job.New(uuid.NewString(), req)
	if err := a.Store.Create(r.Context(), j); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save job")
		return
	}
	if err := a.Queue.Enqueue(r.Context(), j.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}
	metrics.WebhookIngested.Inc()
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": "accepted",
		"job_id": j.ID,
	})
}

func (a *API) latestReconciliation(w http.ResponseWriter, r *http.Request) {
	run, err := a.Store.LatestReconciliationRun(r.Context())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "no runs yet")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load run")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	j, err := a.Store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load job")
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (a *API) enqueueNewJob(w http.ResponseWriter, r *http.Request, req job.CreateRequest) error {
	j := job.New(uuid.NewString(), req)
	if err := a.Store.Create(r.Context(), j); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save job")
		return err
	}
	if err := a.Queue.Enqueue(r.Context(), j.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue job")
		return err
	}
	writeJSON(w, http.StatusAccepted, j)
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
