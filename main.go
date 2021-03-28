package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"net/http"
	"time"
)

type DockerHandle struct {
	dockerClient *client.Client
}

func (dockerPtr *DockerHandle) home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hey")
}

type ContainerResponse struct {
	Id string `json:"id"`
	Image string `json:"image"`
}

type ServiceResponse struct {
	ImageName string `json:"image_name"`
	ImageNameSHA string `json:"image_name_hash"`
	CurrentReplicas uint64 `json:"current_replicas"`
	MaxReplicas uint64 `json:"max_replicas"`
	LastUpdated time.Time `json:"last_updated"`
	IsGlobal bool `json:"is_global"`
}

func (dockerPtr *DockerHandle) containerList(w http.ResponseWriter, _ *http.Request) {
	containers, err := dockerPtr.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	var containersList []ContainerResponse
	for _, container := range containers {
		containersList = append(containersList, ContainerResponse{ Id: container.ID, Image: container.Image })
	}
	err = json.NewEncoder(w).Encode(containersList)
}

func (dockerPtr *DockerHandle) serviceList(w http.ResponseWriter, _ *http.Request) {
	services, err := dockerPtr.dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
	tasks, err := dockerPtr.dockerClient.TaskList(context.Background(), types.TaskListOptions{})
	if err != nil {
		panic(err)
	}

	runningTasks := make(map[string]uint64)

	for _, task := range tasks {
		if task.Status.State == swarm.TaskStateRunning {
			runningTasks[task.ServiceID]++
		}
	}

	var serviceList []ServiceResponse
	for _, service := range services {
		var currentReplicas uint64 = 0
		var maxReplicas uint64 = 1
		isGlobal := true
		if service.Spec.Mode.Replicated != nil {
			if replicas, ok := runningTasks[service.ID]; ok {
				currentReplicas = replicas
			}
			maxReplicas = *service.Spec.Mode.Replicated.Replicas
			isGlobal = false
		}
		serviceResponse := ServiceResponse{
			ImageName: service.Spec.Annotations.Name,
			ImageNameSHA: service.Spec.TaskTemplate.ContainerSpec.Image,
			CurrentReplicas: currentReplicas,
			MaxReplicas: maxReplicas,
			LastUpdated: service.Meta.UpdatedAt,
			IsGlobal: isGlobal,
		}
		serviceList = append(serviceList, serviceResponse)
	}
	err = json.NewEncoder(w).Encode(serviceList)
}

func main() {
	cli, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		panic(err)
	}

	dockerClient := DockerHandle{dockerClient: cli}
	http.Handle("/", http.HandlerFunc(dockerClient.home))
	http.Handle("/containers", http.HandlerFunc(dockerClient.containerList))
	http.Handle("/services", http.HandlerFunc(dockerClient.serviceList))

	err = http.ListenAndServe(":8888", nil)
}