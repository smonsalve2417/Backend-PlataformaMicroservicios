package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

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
