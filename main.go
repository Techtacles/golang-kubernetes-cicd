package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	client *kubernetes.Clientset
	err    error
)

func main() {
	ctx := context.Background()
	if client, err = get_client(); err != nil {
		fmt.Println("Error %s", err)
		os.Exit(1)
	}
	// fmt.Printf("Client %s", client)
	deployment_labels, err := deploy(ctx, client)
	if err != nil {
		fmt.Println("Error creating deployment journals %s", err)
	}

	fmt.Printf("The deployment label is  %s", deployment_labels)
	err = wait_for_pods(ctx, client, deployment_labels)
	fmt.Print()

}
func get_client() (*kubernetes.Clientset, error) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset, nil

}

func deploy(ctx context.Context, client *kubernetes.Clientset) (map[string]string, error) {
	// deployments takes a namespace. Here, we are using the default namespace
	// we inspect the the definition of Create method and we see it takes in metav1.CreateOptions.
	// So we import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" and use it(from source definition)
	app_file, err := os.ReadFile("app.yaml")
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s", err)
	}
	runtime_obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(app_file, nil, nil)
	new_dep := runtime_obj.(*v1.Deployment) // the same with &v1.Deployment{} but this time, for the yaml

	// in the next line, we check if the deployment name already exists.
	_, err = client.AppsV1().Deployments("default").Get(ctx, new_dep.Name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		// If deployment does not exist, create it
		deployment_response, err := client.AppsV1().Deployments("default").Create(ctx, new_dep, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("deployment error %s", err)

		}
		return deployment_response.Labels, nil

	} else if err != nil && !errors.IsNotFound(err) {
		fmt.Errorf("deployment alrady exists %s", err)

	}
	// if found, update it
	deployment_response, err := client.AppsV1().Deployments("default").Update(ctx, new_dep, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("deployment error %s", err)

	}
	return deployment_response.Spec.Template.Labels, nil

}

func wait_for_pods(ctx context.Context, client *kubernetes.Clientset, deployment_labels map[string]string) error {
	for {
		label_selector, err := labels.ValidatedSelectorFromSet(deployment_labels)
		if err != nil {
			fmt.Errorf("unable to get labels from deployment %s", err)
		}
		pod_list, err := client.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
			LabelSelector: label_selector.String(),
		})
		if err != nil {
			fmt.Errorf("failed to list pods %s", err)
		}
		podsRunning := 0
		// iterate through the pods and get the pods not running.
		for _, pod := range pod_list.Items {
			if pod.Status.Phase == "Running" {
				podsRunning++
			}

		}
		fmt.Printf("Waiting for pods to run. Running %d out of %d", podsRunning, len(pod_list.Items))
		// stop if all pods are running
		if podsRunning == len(pod_list.Items) {
			break
		}
		time.Sleep(5 * time.Second)

	}

	return nil

}
