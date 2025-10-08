package main

//Son todas las funciones que interactúan con la base de datos y Docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type store struct {
	mongoClient *mongo.Client
	database    *mongo.Database
	client      *client.Client
}

func NewStore(mongoClient *mongo.Client, client *client.Client) *store {
	return &store{mongoClient: mongoClient, database: mongoClient.Database(mongoDatabaseName), client: client}
}

var (
	ctx = context.Background()
)

func (s *store) NewContainer(containerImage, serviceName string) error {
	ctx := context.Background()

	if !strings.Contains(containerImage, ":") {
		containerImage = containerImage + ":latest"
	}

	// Verificar si el contenedor ya está corriendo
	isRunning, _ := s.IsContainerRunning(serviceName)
	if isRunning {
		return fmt.Errorf("container %s is already running", serviceName)
	}

	// Verificar si la imagen existe localmente
	exists, err := s.ImageExists(containerImage)
	if err != nil {
		log.Fatalf("Error checking image existence: %v", err)
		return err
	}

	if !exists {
		log.Printf("Image %s not found, pulling from Docker Hub...", containerImage)
		if err := s.ImagePull(containerImage); err != nil {
			log.Fatalf("Error pulling image: %v", err)
			return err
		}
		log.Printf("Image %s pulled successfully", containerImage)
	}

	// Definir puertos expuestos (el microservicio escucha en 8000)
	portSet := nat.PortSet{
		"8000/tcp": struct{}{},
	}

	// Crear contenedor con labels para Traefik
	resp, err := s.client.ContainerCreate(ctx, &container.Config{
		Image:        containerImage,
		ExposedPorts: portSet,
		Env: []string{
			"MICROSERVICIO_NAME=" + serviceName, // <--- aquí pasamos la variable
		},
		Labels: map[string]string{
			"traefik.enable": "true",
			"traefik.http.routers." + serviceName + ".rule":                      "PathPrefix(`/" + serviceName + "`)",
			"traefik.http.services." + serviceName + ".loadbalancer.server.port": "8000",
		},
	}, &container.HostConfig{
		NetworkMode: "backend-network", // usa la misma red que Traefik
	}, nil, nil, serviceName)
	if err != nil {
		log.Fatal(err)
		return err
	}

	// Iniciar contenedor
	if err := s.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Fatal(err)
		return err
	}

	fmt.Printf("Contenedor '%s' iniciado con ID: %s\n", serviceName, resp.ID)
	return nil
}

func (s *store) IsContainerRunning(containerName string) (bool, error) {
	ctx := context.Background()

	// Obtener estado del contenedor
	containerJSON, err := s.client.ContainerInspect(ctx, containerName)
	if err != nil {
		return false, err
	}

	fmt.Println("Estado del contenedor:", containerJSON.State.Status)

	// Retorna si está en estado "running"
	return containerJSON.State.Running, nil
}

func (s *store) ImagePull(containerImage string) (err error) {
	ioReader, err := s.client.ImagePull(ctx, containerImage, image.PullOptions{})
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer ioReader.Close()

	_, err = io.Copy(os.Stdout, ioReader)
	if err != nil {
		log.Fatalf("Error leyendo salida: %v", err)
		return err
	}
	return nil
}

func (s *store) ImageExists(imageName string) (bool, error) {
	ctx := context.Background()
	images, err := s.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		log.Fatalf("Error al listar imágenes: %v", err)
		return false, err
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *store) BuildContainerImage(workspaceDir string, imageName string) error {
	// Crear tar del workspace
	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)

	err := filepath.Walk(workspaceDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(workspaceDir, file)
		if err != nil {
			return err
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		hdr := &tar.Header{
			Name: relPath,
			Mode: 0644,
			Size: fi.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error creando tar: %v", err)
	}
	tw.Close()

	// Construir imagen con Docker
	buildOptions := build.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile", // Debe existir en workspaceDir
		Remove:     true,         // Limpiar capas intermedias
	}

	buildResp, err := s.client.ImageBuild(ctx, bytes.NewReader(tarBuf.Bytes()), buildOptions)
	if err != nil {
		return fmt.Errorf("error construyendo imagen: %v", err)
	}
	defer buildResp.Body.Close()

	// Mostrar salida de build
	if _, err := io.Copy(os.Stdout, buildResp.Body); err != nil {
		return fmt.Errorf("error leyendo salida build: %v", err)
	}

	fmt.Println("✅ Imagen construida con nombre:", imageName)
	return nil
}

func (s *store) StopAndRemoveContainer(containerName string) error {

	// Try stopping the container (ignore if it's not running)
	if err := s.client.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		if !client.IsErrNotFound(err) {
			return fmt.Errorf("failed to stop container %s: %w", containerName, err)
		}
	}

	// Remove the container (force = true ensures cleanup even if stopped fails)
	if err := s.client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerName, err)
	}

	return nil
}

func (s *store) StopContainer(containerName string) error {

	if err := s.client.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("container %s not found", containerName)
		}
		return fmt.Errorf("failed to stop container %s: %w", containerName, err)
	}

	return nil
}

func (s *store) ListContainers() ([]container.Summary, error) {
	ctx := context.Background()

	// false -> incluye parados y corriendo
	containers, err := s.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}

func (s *store) StartContainer(containerName string) error {
	ctx := context.Background()

	if err := s.client.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("container %s not found", containerName)
		}
		return fmt.Errorf("failed to start container %s: %w", containerName, err)
	}

	return nil
}

func (s *store) SaveContainer(record ContainerRecord) (primitive.ObjectID, error) {
	collection := s.database.Collection("containers")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, record)
	if err != nil {
		return primitive.NilObjectID, err
	}

	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("inserted document ID is not an ObjectID")
	}

	return insertedID, nil
}

func (s *store) UpdateContainerStatus(userID string, containerName string, status bool) error {
	collection := s.database.Collection("containers")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"userId":        userID,
		"containerName": containerName,
	}

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no container found with name %s for user %s", containerName, userID)
	}

	return nil
}

func (s *store) ContainerExists(name string) (bool, error) {
	containers, err := s.ListContainers()
	if err != nil {
		return false, err
	}

	for _, c := range containers {
		// Names en Docker vienen con un "/" al inicio (ej: "/mi-contenedor")
		for _, containerName := range c.Names {
			if containerName == "/"+name {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *store) DeleteContainerDocument(userID, containerName string) error {
	collection := s.database.Collection("containers")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"userId":        userID,
		"containerName": containerName,
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete container document: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no document found with userID=%s and containerName=%s", userID, containerName)
	}

	return nil
}

func (s *store) GetContainersByUser(userID string) ([]ContainerRecord, error) {
	collection := s.database.Collection("containers")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"userId": userID}
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to querry containers: %w", err)
	}

	defer cur.Close(ctx)

	var results []ContainerRecord
	for cur.Next(ctx) {
		var rec ContainerRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("failed to decode container : %w", err)
		}
		results = append(results, rec)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return results, nil
}

// Funcion para repetir asyncrono
func (s *store) GetAllContainers() ([]ContainerRecord, error) {
	collection := s.database.Collection("containers")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query containers: %w", err)
	}
	defer cur.Close(ctx)

	var results []ContainerRecord
	for cur.Next(ctx) {
		var rec ContainerRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("failed to decode container: %w", err)
		}
		results = append(results, rec)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return results, nil
}

func (s *store) SaveUpdate(update ContainerUpdate) (primitive.ObjectID, error) {
	collection := s.database.Collection("history")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := collection.InsertOne(ctx, update)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to insert update: %w", err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("inserted document ID is not an ObjectID")
	}
	return oid, nil
}

func (s *store) GetHistoryByUser(userID string) ([]ContainerUpdate, error) {
	collection := s.database.Collection("history")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"userId": userID}
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to querry containers: %w", err)
	}

	defer cur.Close(ctx)

	var results []ContainerUpdate
	for cur.Next(ctx) {
		var rec ContainerUpdate
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("failed to decode container : %w", err)
		}
		results = append(results, rec)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return results, nil
}

func (s *store) GetAllContainersHistory() ([]ContainerUpdate, error) {
	collection := s.database.Collection("history")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query containers: %w", err)
	}
	defer cur.Close(ctx)

	var results []ContainerUpdate
	for cur.Next(ctx) {
		var rec ContainerUpdate
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("failed to decode container: %w", err)
		}
		results = append(results, rec)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return results, nil
}

func (s *store) GetLastHistory() (*ContainerUpdate, error) {
	collection := s.database.Collection("history")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	var result ContainerUpdate
	err := collection.FindOne(ctx, bson.M{}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch last history: %w", err)
	}

	return &result, nil
}
