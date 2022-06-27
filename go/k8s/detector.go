package k8s

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var Pods []v1.Pod

func Scan(clientset *kubernetes.Clientset, namespacePrefix string) {
	log.Debug().Msg("Fetching list of namespaces")

	// Fetch all namespaces with prefix based on namespacePrefix
	filteredNamespaces, err := getFilteredNamespaces(clientset, namespacePrefix)
	if err != nil {
		log.Fatal().Err(err).Msg("could not fetch list of namespaces")
	}

	// Collect the pods for each filtered namespace
	Pods = []v1.Pod{}
	for _, ns := range filteredNamespaces {
		Pods = append(Pods, getPods(clientset, ns).Items...)
	}

	log.Debug().
		Int("numPods", len(Pods)).
		Msgf("There are %d pods in namespaces with prefix %s", len(Pods), namespacePrefix)
	for _, pod := range Pods {
		log.Debug().
			Str("name", pod.GetName()).
			Str("ip", pod.Status.PodIP).
			Str("host", pod.Status.HostIP).
			Str("namespace", pod.Namespace).
			Msg("Detected pod")
	}
}

// Fetch a list of all namespaces in the cluster with the given prefix
func getFilteredNamespaces(clientset *kubernetes.Clientset, namespacePrefix string) ([]string, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	filteredNamespaces := []string{}
	for _, ns := range namespaces.Items {
		log.Debug().Str("namespace", ns.GetName()).Msg("Detected namespace")

		if strings.HasPrefix(ns.GetName(), namespacePrefix) {
			filteredNamespaces = append(filteredNamespaces, ns.GetName())
		}
	}

	return filteredNamespaces, nil

}

func PodsExcluding(pod *v1.Pod) []v1.Pod {
	var filtered []v1.Pod = []v1.Pod{}

	for _, p := range Pods {
		if pod.Status.PodIP != p.Status.PodIP {
			filtered = append(filtered, p)
		}
	}

	return filtered

}

// Fetch a list of all pods in the given namespace
func getPods(clientset *kubernetes.Clientset, ns string) *v1.PodList {
	pods, err := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return pods
}
