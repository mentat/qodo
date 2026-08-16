package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

// InvoiceHandler adapts services.InvoiceService to HTTP.
type InvoiceHandler struct {
	svc *services.InvoiceService
}

// NewInvoiceHandlerWithService lets main and tests inject a service instance.
func NewInvoiceHandlerWithService(svc *services.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

// defaultDelegatedLimitCents is the per-submitter approval limit used when the
// caller does not supply one: 5,000.00 in minor units.
const defaultDelegatedLimitCents int64 = 500000

func invoiceStatusFor(err error) (int, string) {
	switch {
	case errors.Is(err, services.ErrUnauthenticated):
		return http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, services.ErrInvoiceNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, services.ErrInvoiceInvalid):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, services.ErrAlreadyDecided):
		return http.StatusConflict, "already decided"
	case errors.Is(err, services.ErrPaymentDeclined):
		return http.StatusBadGateway, "payment declined"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

// Create stores a new invoice and routes it for approval.
func (h *InvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	var inv services.Invoice
	if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	limit := defaultDelegatedLimitCents
	if raw := r.URL.Query().Get("limitCents"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			limit = parsed
		}
	}

	created, err := h.svc.Create(r.Context(), uid, inv, limit)
	if err != nil {
		status, msg := invoiceStatusFor(err)
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Get returns one invoice belonging to the current user.
func (h *InvoiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	inv, err := h.svc.Get(r.Context(), uid, chi.URLParam(r, "invoiceID"))
	if err != nil {
		status, msg := invoiceStatusFor(err)
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// Decide records an approver's decision on a pending invoice.
func (h *InvoiceHandler) Decide(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	var body struct {
		ApproverID string `json:"approverId"`
		Decision   string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Decision != "approved" && body.Decision != "rejected" {
		writeError(w, http.StatusBadRequest, "decision must be approved or rejected")
		return
	}

	inv, err := h.svc.Decide(r.Context(), uid, chi.URLParam(r, "invoiceID"), body.ApproverID, body.Decision)
	if err != nil {
		status, msg := invoiceStatusFor(err)
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// Pay releases funds for an approved invoice.
func (h *InvoiceHandler) Pay(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	inv, err := h.svc.Pay(r.Context(), uid, chi.URLParam(r, "invoiceID"))
	if err != nil {
		status, msg := invoiceStatusFor(err)
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}
