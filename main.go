package main

import (
	"context"
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

	mongoClient, err := NewMongoDBStorage(mongoAddr, mongoDatabaseName)
	if err != nil {
		log.Fatal("Error connecting to MongoDB: ", err)
	}

	// Docker
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("could not create Docker client: %v", err)
	}

	ctx := context.Background()
	_, err = dockerClient.Ping(ctx)
	if err != nil {
		go func() {
			dockerPath := `C:\Program Files\Docker\Docker\Docker Desktop.exe`
			cmd := exec.Command(dockerPath)
			if err := cmd.Start(); err != nil {
				fmt.Println("Error starting Docker Desktop:", err)
				return
			}
		}()
	}

	if err := WaitForDocker(dockerClient, 10*time.Second); err != nil {
		fmt.Println("Docker does not seem to be running.")
		fmt.Println("Make sure Docker Desktop (or dockerd) is started and run this program again.")
		return
	}

	fmt.Println("✅ Docker daemon is running!")

	store := NewStore(mongoClient.GetDatabase().Client(), dockerClient)
	handler := NewHandler(store)
	handler.registerRoutes(mux)

	// ✅ Apply CORS middleware
	corsMux := enableCORS(mux)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go StartPeriodically(ctx, store, 60*time.Second)

	log.Printf("Starting HTTP server at %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, corsMux); err != nil {
		log.Fatal("Failed to start http server:", err)
	}
}

func GetEnv(key string, fallback string) string {
	if value, ok := syscall.Getenv(key); ok {
		return value
	}
	return fallback
}

// ✅ CORS Middleware
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Adjust the origin as needed
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func StartContainersWithDB(ctx context.Context, s *store) {
	records, err := s.GetAllContainers()
	if err != nil {
		log.Printf("[startup] error leyendo containers :%v", err)
		return
	}

	started := 0
	skipped := 0

	for _, rec := range records {
		if !rec.Status {
			skipped++
			continue
		}

		running, err := s.IsContainerRunning(rec.ContainerName)
		if err != nil {
			log.Printf("[startup] error consultando %s: %v", rec.ContainerName, err)
			continue
		}
		if running {
			skipped++
			continue
		}
		if err := s.StartContainer(rec.ContainerName); err != nil {
			log.Printf("[startup] no se pudo iniciar %s: %v", rec.ContainerName, err)
			continue
		}
		started++
		log.Printf("[startup] iniciado %s", rec.ContainerName)
	}
	log.Printf("[startup] verificación inicial terminada. iniciados=%d, omitidos=%d, total=%d", started, skipped, len(records))
}

func StartPeriodically(ctx context.Context, s *store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			StartContainersWithDB(ctx, s)
		case <-ctx.Done():
			log.Println("[startup] bucle detenido")
			return
		}
	}
}
