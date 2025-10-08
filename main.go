package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/docker/docker/client"
)

var (
	httpAddr          = ":8080"
	mongoAddr         = GetEnv("MONGO_ADDR", "mongodb://root:example@mongodb:27017/")
	mongoDatabaseName = GetEnv("MONGO_DATABASE_NAME", "microservicios")
)

const url = "https://roble-api.openlab.uninorte.edu.co"
const proyectID = "actividad_plataforma_de_microservicios_c12ce95f80"

func main() {

	fmt.Println("MONGO_ADDR:", mongoAddr)
	fmt.Println("MONGO_DATABASE_NAME:", mongoDatabaseName)

	mux := http.NewServeMux()

	mongoCLient, err := NewMongoDBStorage(mongoAddr, mongoDatabaseName)
	if err != nil {
		log.Fatal("Error connecting to MongoDB: ", err)
	}

	//docker
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("could not create Docker client: %v", err)
	}

	_, err = dockerClient.Ping(ctx)
	if err != nil {
		go func() {
			dockerPath := `C:\Program Files\Docker\Docker\Docker Desktop.exe`

			// Start Docker Desktop
			cmd := exec.Command(dockerPath)
			err := cmd.Start()
			if err != nil {
				fmt.Println("Error starting Docker Desktop:", err)
				return
			}
		}()

	}

	// Ping Docker daemon to check if it's running
	// Wait up to 10 seconds for Docker
	if err := WaitForDocker(dockerClient, 10*time.Second); err != nil {
		fmt.Println("Docker does not seem to be running.")
		fmt.Println("Make sure Docker Desktop (or dockerd) is started and run this program again.")
		return
	}

	// If we reach here, Docker is alive
	fmt.Println("✅ Docker daemon is running!")

	//"libreria" de funciones a utilizar
	store := NewStore(mongoCLient.GetDatabase().Client(), dockerClient)

	//rutas del servidor http
	handler := NewHandler(store)
	handler.registerRoutes(mux)
	log.Printf("Starting HTTP server at %s", httpAddr)

	//funcion asyncrona que recargue los microservicios y sus estados

	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatal("Failed to start http server", err)
	}

}

func GetEnv(key string, fallback string) string {
	if value, ok := syscall.Getenv(key); ok {
		return value
	}

	return fallback
}
