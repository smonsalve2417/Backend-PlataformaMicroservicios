package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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
	mux.HandleFunc("POST /new/image", h.HandleImageCreation)
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

func (h *handler) HandleImageCreation(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Recibido solicitud de creación de imagen")
	// Limitar tamaño máximo de archivos
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20 MB
		log.Println("Nerror en ParseMultipartForm:", err)
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	fmt.Println("ParseMultipartForm exitoso")

	// Leer nombre del servicio
	name := r.FormValue("name")
	if name == "" {
		log.Println("Nombre del servicio es obligatorio")
		WriteError(w, http.StatusBadRequest, "nombre del servicio es obligatorio")
		return
	}

	fmt.Println("Nombre del servicio:", name)

	// Crear carpeta workspace
	workspaceDir := "./workspace/" + name
	os.MkdirAll(workspaceDir, os.ModePerm)

	// Leer todos los archivos de código (ej: campo "files")
	files := r.MultipartForm.File["files"]
	for _, fHeader := range files {
		file, err := fHeader.Open()
		if err != nil {
			log.Printf("Error abriendo archivo %s: %v", fHeader.Filename, err)
			continue
		}
		defer file.Close()

		dst, _ := os.Create(workspaceDir + "/" + fHeader.Filename)
		io.Copy(dst, file)
		dst.Close()
	}

	// Leer Dockerfile (opcional)
	dockerFileHeader, ok := r.MultipartForm.File["dockerfile"]
	if ok && len(dockerFileHeader) > 0 {
		dockerFile, _ := dockerFileHeader[0].Open()
		defer dockerFile.Close()
		dstDocker, _ := os.Create(workspaceDir + "/Dockerfile")
		io.Copy(dstDocker, dockerFile)
		dstDocker.Close()
	}
	// Construir imagen desde workspace
	imageName := name + ":latest"
	if err := h.store.BuildContainerImage(workspaceDir, imageName); err != nil {
		fmt.Printf("Error al construir imagen: %v", err)
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"image": imageName})
}
