package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var listenAddress = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var pingFrequency = flag.Int("ping-frequency", 5, "How often pings are performed.")
var namespacePrefix = flag.String("namespace-prefix", "netpong-", "The namespace prefix to watch. Defaults to netpong-")
var kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
var debug = flag.Bool("debug", false, "sets log level to debug")

func main() {

	// Read config from env variables or flags
	flag.Parse()

	log.Warn().Bool("debug", *debug).Msg("Debugging enabled")

	// Default level is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Error building kubeconfig")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("could not get new config")
	}

	// start periodic test
	go testLoop(clientset)

	// start webserver
	http.HandleFunc("/ping", handlePing)

	log.Fatal().Err(http.ListenAndServe(*listenAddress, nil)).Msg("ListenAndServe failed")
}

// Fetch a list of all namespaces in the cluster with the given prefix
func getFilteredNamespaces(clientset *kubernetes.Clientset, namespacePrefix *string) ([]string, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	filteredNamespaces := []string{}
	for _, ns := range namespaces.Items {
		log.Debug().Str("namespace", ns.GetName()).Msg("Detected namespace")

		if strings.HasPrefix(ns.GetName(), *namespacePrefix) {
			filteredNamespaces = append(filteredNamespaces, ns.GetName())
		}
	}

	return filteredNamespaces, nil

}

// Fetch a list of all pods in the given namespace
func getPods(clientset *kubernetes.Clientset, ns string) *v1.PodList {
	pods, err := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return pods
}

func testLoop(clientset *kubernetes.Clientset) {
	for {
		log.Debug().Msg("Performing ping test")

		// Fetch all namespaces with prefix based on flag (default netpong-)
		filteredNamespaces, err := getFilteredNamespaces(clientset, namespacePrefix)
		if err != nil {
			log.Fatal().Err(err).Msg("could not fetch list of namespaces")
		}

		// Collect the pods for each filtered namespace
		pods := []v1.Pod{}
		for _, ns := range filteredNamespaces {
			pods = append(pods, getPods(clientset, ns).Items...)
		}

		log.Debug().
			Int("numPods", len(pods)).
			Msgf("There are %d pods in the cluster", len(pods))
		for _, pod := range pods {
			log.Debug().
				Str("name", pod.GetName()).
				Str("ip", pod.Status.PodIP).
				Str("host", pod.Status.HostIP).
				Msg("Detected pod")
		}

		if len(pods) > 0 {
			// Select random pod and ping it
			dest := pods[rand.Intn(len(pods))]
			go testPing(dest)
		}

		time.Sleep(time.Duration(*pingFrequency) * time.Second)
	}
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("Ping received")
	w.Write([]byte("pong"))
}

func testPing(pod v1.Pod) {
	url := fmt.Sprintf("http://%s:8080/ping", pod.Status.PodIP)

	log.Debug().Msgf("Sending ping to %s", url)

	_, err := testHTTP(url)
	if err != nil {
		log.
			Warn().
			Err(err).
			Str("url", url).
			Msg("Test ping failed")
	}
}

func testHTTP(url string) (time.Time, error) {

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating HTTP request")
	}

	ctx := req.Context()
	req = req.WithContext(ctx)
	// Send request by default HTTP client
	client := http.DefaultClient
	client.Timeout = time.Duration(2) * time.Second
	res, err := client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		return time.Time{}, err
	}
	res.Body.Close()
	end := time.Now()
	return end, nil
}
