package handlers

import (
	"encoding/json"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
	"github.com/mentat/qodo/api/middleware"
	"github.com/mentat/qodo/api/services"
)

// Invoice is re-exported for callers that reference handlers.Invoice.
type Invoice = services.Invoice

// InvoiceHandler adapts services.InvoiceService to HTTP.
type InvoiceHandler struct {
	svc *services.InvoiceService
}

// NewInvoiceHandler constructs a handler using a Firestore-backed service.
func NewInvoiceHandler(fs *firestore.Client) *InvoiceHandler {
	return &InvoiceHandler{svc: services.NewInvoiceService(fs)}
}

// NewInvoiceHandlerWithService lets tests inject a service instance (e.g.
// pointing at a per-test collection).
func NewInvoiceHandlerWithService(svc *services.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

// List returns the current user's invoices. Query params: status, entityId.
func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	filter := services.InvoiceFilter{
		Status:   services.InvoiceStatus(r.URL.Query().Get("status")),
		EntityID: r.URL.Query().Get("entityId"),
	}

	invoices, err := h.svc.List(r.Context(), uid, filter)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, invoices)
}

// Create adds a new invoice and routes it through the approval policy.
func (h *InvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var in services.Invoice
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	inv, err := h.svc.Create(r.Context(), uid, services.InvoiceCreateInput{
		VendorName:    in.VendorName,
		InvoiceNumber: in.InvoiceNumber,
		EntityID:      in.EntityID,
		CurrencyCode:  in.CurrencyCode,
		AmountCents:   in.AmountCents,
		Lines:         in.Lines,
		DueDate:       in.DueDate,
	})
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

// Get returns a single invoice.
func (h *InvoiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	inv, err := h.svc.Get(r.Context(), uid, id)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// Approve records an approval on the invoice's first pending step.
func (h *InvoiceHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, true)
}

// Reject rejects the invoice. Terminal.
func (h *InvoiceHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, false)
}

func (h *InvoiceHandler) decide(w http.ResponseWriter, r *http.Request, approve bool) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	// Body is optional — an approval note is the only field.
	var body struct {
		Note string `json:"note"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	inv, err := h.svc.Decide(r.Context(), uid, id, approve, body.Note)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// Pay books an approved invoice in the ERP and submits it for payment.
func (h *InvoiceHandler) Pay(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	inv, err := h.svc.SubmitPayment(r.Context(), uid, id)
	if err != nil {
		s, m := statusFor(err)
		writeError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// Policy returns the active approval policy so the client can explain routing
// to the user before they submit.
func (h *InvoiceHandler) Policy(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Policy())
}
