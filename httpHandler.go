package main

//Aquí creo los endpoints
//Crear una función que retorne en JSON los documentos de la colección containers que pertenezcan al usuario autenticado GET

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-playground/validator"
)

type handler struct {
	store *store
}

func NewHandler(store *store) *handler {
	return &handler{store: store}
}

func (h *handler) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/signup", h.HandleUserRegister)
	mux.HandleFunc("POST /auth/login", h.HandleUserLogin)
	mux.HandleFunc("POST /new/container", WithJWTAuth(h.HandleNewContainer))
	mux.HandleFunc("POST /remove/container", WithJWTAuth(h.HandleRemoveContainer))
	mux.HandleFunc("POST /stop/container", WithJWTAuth(h.HandleStopContainer))
	mux.HandleFunc("POST /start/container", WithJWTAuth(h.HandleStartContainer))
	mux.HandleFunc("POST /new/image", WithJWTAuth(h.HandleImageCreation))
	mux.HandleFunc("GET /containers", WithJWTAuth(h.HandleListUserContainers))
}

func (h *handler) HandleUserRegister(w http.ResponseWriter, r *http.Request) {
	var payload registerUser

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

	// Marshal a JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling payload:", err)
		return
	}

	// Send POST request
	resp, err := http.Post(url+"/auth/"+proyectID+"/signup-direct", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Body:", payload)

	// Read response
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status:", resp.Status)
	fmt.Println("Response:", string(body))

	if resp.StatusCode != http.StatusOK {
		WriteError(w, resp.StatusCode, string(body))
		return
	}

	WriteJSON(w, http.StatusOK, string(body))

}

func (h *handler) HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	var payload loginUser

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

	// Marshal a JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling payload:", err)
		return
	}

	// Send POST request
	resp, err := http.Post(url+"/auth/"+proyectID+"/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Body:", payload)

	// Read response
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status:", resp.Status)
	fmt.Println("Response:", string(body))

	// Extract only the tokens
	var response loginResponse
	if err := json.Unmarshal(body, &response); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to parse login response")
		return
	}

	o := loginResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
	}

	if resp.StatusCode != http.StatusCreated {
		WriteError(w, resp.StatusCode, string(body))
		return
	}

	WriteJSON(w, http.StatusOK, o)

}

func (h *handler) HandleNewContainer(w http.ResponseWriter, r *http.Request) {
	//Same
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Unauthorized access: %v", err)
		WriteError(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
		return
	}

	println("User ID from context:", userID)
	//

	var payload contenedor //Tipo del JSON que se recibe en el ENDPOINT

	//Same
	if err := ParseJSON(r, &payload); err != nil { //Parsea el JSON al struct
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
	//

	//-Logica del ENDPOINT------->
	err = h.store.NewContainer(payload.Image, payload.Image)
	o := payload
	if err != nil {
		WriteError(w, http.StatusConflict, err.Error())
		return
	}

	// Guardar el contenedor en MongoDB

	status, err := h.store.IsContainerRunning(payload.Image)
	println("Container status:", status)
	if err != nil {
		WriteError(w, http.StatusConflict, err.Error())
		return
	}

	record := ContainerRecord{
		UserID:        userID,
		ContainerName: payload.Image,
		Status:        status,
		CreatedAt:     time.Now(),
	}

	_, err = h.store.SaveContainer(record)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to save container: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, o)
	//<----------------------

}

func (h *handler) HandleRemoveContainer(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Unauthorized access: %v", err)
		WriteError(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
		return
	}
	println("User ID from context:", userID)

	var payload contenedor

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

	err = h.store.StopAndRemoveContainer(payload.Image)
	o := payload
	if err != nil {
		WriteError(w, http.StatusConflict, err.Error())
		return
	}

	err = h.store.DeleteContainerDocument(userID, payload.Image)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to update container status: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, o)

}

func (h *handler) HandleStopContainer(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Unauthorized access: %v", err)
		WriteError(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
		return
	}
	println("User ID from context:", userID)

	var payload contenedor

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

	//validacion de propiedad

	err = h.store.StopContainer(payload.Image)
	o := payload
	if err != nil {
		WriteError(w, http.StatusConflict, err.Error())
		return
	}

	err = h.store.UpdateContainerStatus(userID, payload.Image, false)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to update container status: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, o)

}

func (h *handler) HandleStartContainer(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Unauthorized access: %v", err)
		WriteError(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
		return
	}
	println("User ID from context:", userID)

	var payload contenedor

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

	//validacion de propiedad

	err = h.store.StartContainer(payload.Image)
	o := payload
	if err != nil {
		WriteError(w, http.StatusConflict, err.Error())
		return
	}

	err = h.store.UpdateContainerStatus(userID, payload.Image, true)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to update container status: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, o)

}

func (h *handler) HandleImageCreation(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Unauthorized access: %v", err)
		WriteError(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
		return
	}
	println("User ID from context:", userID)

	fmt.Println("Recibido solicitud de creación de imagen")

	// Limitar tamaño máximo de archivos
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20 MB
		log.Println("Error en ParseMultipartForm:", err)
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

	exists, err := h.store.ContainerExists(name)
	if err != nil {
		log.Printf("Error checking container existence: %v", err)
		WriteError(w, http.StatusInternalServerError, "error checking container existence")
		return
	}
	if exists {
		log.Printf("Container with name %s already exists", name)
		WriteError(w, http.StatusConflict, "container with this name already exists")
		return
	}

	// Crear carpeta workspace
	workspaceDir := "./workspace/" + name
	os.MkdirAll(workspaceDir, os.ModePerm)

	// Leer archivo app.py
	appFileHeader, ok := r.MultipartForm.File["app"]
	if !ok || len(appFileHeader) == 0 {
		log.Println("Archivo app.py es obligatorio")
		WriteError(w, http.StatusBadRequest, "archivo app.py es obligatorio")
		return
	}

	appFile, err := appFileHeader[0].Open()
	if err != nil {
		log.Printf("Error abriendo app.py: %v", err)
		WriteError(w, http.StatusInternalServerError, "no se pudo abrir app.py")
		return
	}
	defer appFile.Close()

	dstApp, err := os.Create(workspaceDir + "/app.py")
	if err != nil {
		log.Printf("Error creando app.py en workspace: %v", err)
		WriteError(w, http.StatusInternalServerError, "no se pudo guardar app.py")
		return
	}
	defer dstApp.Close()
	io.Copy(dstApp, appFile)

	// Copiar Dockerfile desde ./files
	srcDocker, err := os.Open("./files/Dockerfile")
	if err != nil {
		log.Printf("Error abriendo Dockerfile por defecto: %v", err)
		WriteError(w, http.StatusInternalServerError, "no se pudo usar Dockerfile por defecto")
		return
	}
	defer srcDocker.Close()

	dstDocker, err := os.Create(workspaceDir + "/Dockerfile")
	if err != nil {
		log.Printf("Error creando Dockerfile en workspace: %v", err)
		WriteError(w, http.StatusInternalServerError, "no se pudo copiar Dockerfile por defecto")
		return
	}
	defer dstDocker.Close()
	io.Copy(dstDocker, srcDocker)

	// Copiar server.py desde ./files
	srcServer, err := os.Open("./files/server.py")
	if err != nil {
		log.Printf("Error abriendo server.py: %v", err)
		WriteError(w, http.StatusInternalServerError, "no se pudo copiar server.py")
		return
	}
	defer srcServer.Close()

	dstServer, err := os.Create(workspaceDir + "/server.py")
	if err != nil {
		log.Printf("Error creando server.py en workspace: %v", err)
		WriteError(w, http.StatusInternalServerError, "no se pudo guardar server.py")
		return
	}
	defer dstServer.Close()
	io.Copy(dstServer, srcServer)

	// Construir imagen desde workspace
	imageName := name + ":latest"
	if err := h.store.BuildContainerImage(workspaceDir, imageName); err != nil {
		fmt.Printf("Error al construir imagen: %v", err)
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"image": imageName})
}

func (h *handler) HandleListUserContainers(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Unauthorized access: %v", err)
		WriteError(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
		return
	}

	println("User ID from context:", userID)

	//
	records, err := h.store.GetContainersByUser(userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to fetch containers: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"containers": records,
		"count":      len(records),
	})

}
