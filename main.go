package main

import (
	"context"
	kube "devops/init/kubernetes"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
)

var (
	client *kubernetes.Clientset
	err    error
)

func main() {
	ctx := context.Background()
	if client, err = kube.GetClient(); err != nil {
		fmt.Println("Error %s", err)
		os.Exit(1)
	}
	// fmt.Printf("Client %s", client)
	basefilepath, _ := os.Getwd()
	full_filepath := filepath.Join(basefilepath, "manifests/app.yaml")
	deployment_labels, err := kube.Deploy(ctx, client, full_filepath)
	if err != nil {
		fmt.Println("Error creating deployment journals %s", err)
	}

	fmt.Printf("The deployment label is  %s", deployment_labels)
	err = kube.WaitForPods(ctx, client, deployment_labels)
	fmt.Print()

}
