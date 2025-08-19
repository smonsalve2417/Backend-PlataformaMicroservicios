package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/client"
)

var (
	httpAddr = ":80"
)

func main() {

	mux := http.NewServeMux()

	//docker
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("ould not create Docker client: %v", err)
	}

	// Ping Docker daemon to check if it's running
	// Wait up to 10 seconds for Docker
	if err := WaitForDocker(cli, 10*time.Second); err != nil {
		fmt.Println("Docker does not seem to be running.")
		fmt.Println("Make sure Docker Desktop (or dockerd) is started and run this program again.")
		return
	}

	// If we reach here, Docker is alive
	fmt.Println("✅ Docker daemon is running!")

	//"libreria" de funciones a utilizar
	store := NewStore(cli)

	//rutas del servidor http
	handler := NewHandler(store)
	handler.registerRoutes(mux)
	log.Printf("Starting HTTP server at %s", httpAddr)

	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatal("Failed to start http server", err)
	}

}
