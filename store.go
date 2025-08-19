package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type store struct {
	client *client.Client
}

func NewStore(client *client.Client) *store {
	return &store{client: client}
}

var (
	ctx = context.Background()
)

func (s *store) StartContainer(containerImage string) (err error) {

	// Verificar si la containerImage existe
	if !ImageExists(s.client, containerImage) {
		log.Printf("Image %s not found, pulling from Docker Hub...", containerImage)
		if err := s.ImagePull(containerImage); err != nil {
			log.Fatalf("Error pulling image: %v", err)
			return err
		}
		log.Printf("Image %s pulled successfully", containerImage)
	}

	// Definir puertos usando nat.PortSet
	portSet := nat.PortSet{
		"80/tcp": struct{}{},
	}

	// Crear contenedor
	resp, err := s.client.ContainerCreate(ctx, &container.Config{
		Image:        containerImage,
		ExposedPorts: portSet,
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "8080"},
			},
		},
	}, &network.NetworkingConfig{}, nil, "mi-nginx")
	if err != nil {
		log.Fatal(err)
	}

	// Iniciar contenedor
	if err := s.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Fatal(err)
		return err
	}

	fmt.Println("Contenedor iniciado con ID:", resp.ID)
	return nil

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

func ImageExists(cli *client.Client, imageName string) bool {
	ctx := context.Background()
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		log.Fatalf("Error al listar imágenes: %v", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				return true
			}
		}
	}
	return false
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
