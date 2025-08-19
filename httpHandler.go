package main

import (
	"log"
	"net/http"

	"github.com/go-playground/validator"
)

type handler struct {
	store *store
}

func NewHandler(store *store) *handler {
	return &handler{store: store}
}

func (h *handler) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /new/container", h.HandleRegister)
}

func (h *handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var payload newDocker

	if err := ParseJSON(r, &payload); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := Validate.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		formattedErrors := FormatValidationErrors(errors)
		WriteError(w, http.StatusBadRequest, "invalid payload: "+formattedErrors)
		return
	}

	err := h.store.StartContainer(payload.Image)
	o := payload
	if err != nil {
		WriteJSON(w, http.StatusConflict, o)
		return
	}

	WriteJSON(w, http.StatusOK, o)

}
