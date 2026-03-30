package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	respond "myevent-back/internal/http"
	"myevent-back/internal/storage"
)

type HealthHandler struct {
	db      *pgxpool.Pool
	storage storage.Provider
}

func NewHealthHandler(db *pgxpool.Pool, storage storage.Provider) *HealthHandler {
	return &HealthHandler{db: db, storage: storage}
}

type healthResponse struct {
	Status   string            `json:"status"`
	Checks   map[string]string `json:"checks"`
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	healthy := true

	// DB check
	if h.db == nil {
		checks["database"] = "skipped"
	} else if err := h.db.Ping(ctx); err != nil {
		checks["database"] = "error: " + err.Error()
		healthy = false
	} else {
		checks["database"] = "ok"
	}

	// R2 check — tenta um PutObject com conteúdo vazio em chave de probe
	if h.storage != nil {
		_, err := h.storage.PutObject(ctx, ".health-probe", []byte{}, "application/octet-stream")
		if err != nil {
			checks["storage"] = "error: " + err.Error()
			healthy = false
		} else {
			// remove o arquivo de probe sem bloquear a resposta
			_ = h.storage.DeleteObject(ctx, ".health-probe")
			checks["storage"] = "ok"
		}
	} else {
		checks["storage"] = "local"
	}

	status := "ok"
	code := http.StatusOK
	if !healthy {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	respond.WriteJSON(w, code, healthResponse{
		Status: status,
		Checks: checks,
	})
}
