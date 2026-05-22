// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/goabonga/infrastructure/internal/domain/resource"
	"github.com/goabonga/infrastructure/internal/secret"
	"github.com/goabonga/infrastructure/internal/state"
)

// SecretHandler serves secrets over HTTP. Unlike the generic handler it never
// returns secret material on list/get; plaintext is only available through the
// explicit reveal endpoint.
type SecretHandler struct {
	svc *secret.Service
}

// NewSecretHandler returns a handler backed by svc.
func NewSecretHandler(svc *secret.Service) *SecretHandler {
	return &SecretHandler{svc: svc}
}

// Register mounts the secret routes under base (e.g. "/api/v1").
func (h *SecretHandler) Register(mux *http.ServeMux, base string) {
	p := base + "/" + resource.KindSecret
	mux.HandleFunc("GET "+p, h.list)
	mux.HandleFunc("GET "+p+"/{uid}", h.get)
	mux.HandleFunc("GET "+p+"/{uid}/reveal", h.reveal)
	mux.HandleFunc("PUT "+p+"/{uid}", h.put)
	mux.HandleFunc("DELETE "+p+"/{uid}", h.delete)
}

func (h *SecretHandler) list(w http.ResponseWriter, _ *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resource.List[resource.SecretSpec, resource.SecretStatus]{
		APIVersion: resource.APIVersion,
		Kind:       resource.KindSecret,
		Items:      items,
	})
}

func (h *SecretHandler) get(w http.ResponseWriter, r *http.Request) {
	sec, err := h.svc.Get(r.PathValue("uid"))
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, sec)
	case errors.Is(err, state.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func (h *SecretHandler) reveal(w http.ResponseWriter, r *http.Request) {
	plaintext, err := h.svc.Reveal(r.PathValue("uid"))
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"data": plaintext})
	case errors.Is(err, state.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func (h *SecretHandler) put(w http.ResponseWriter, r *http.Request) {
	var in resource.Secret
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if in.Spec.Data == "" {
		writeError(w, http.StatusBadRequest, "secret: data is required")
		return
	}
	out, err := h.svc.Put(r.PathValue("uid"), in.Metadata.Name, in.Spec.Data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *SecretHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.PathValue("uid")); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
