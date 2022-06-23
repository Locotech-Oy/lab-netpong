package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/Locotech-Oy/netpong/k8s"
	"github.com/Locotech-Oy/netpong/netpong"

	"flag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var listenAddress = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var pingFrequency = flag.Int("ping-frequency", 5, "How often pings are performed.")
var namespacePrefix = flag.String("namespace-prefix", "netpong", "The namespace prefix to watch. Defaults to netpong-")
var kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
var debug = flag.Bool("debug", false, "sets log level to debug")

var netpongClient *netpong.NetpongClient
var targetPod *v1.Pod

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

	// Resolve pod this code runs in
	var whoami *v1.Pod
	for {
		k8s.Scan(clientset, *namespacePrefix)
		whoami, err = k8s.Whoami(k8s.Pods[:])

		if err != nil {
			log.Fatal().Err(err).Msg("could not resolve whoami pod")
		}

		if whoami.Status.PodIP == "" {
			log.Info().Msg("Pod does not have IP, waiting...")
			time.Sleep(time.Duration(1) * time.Second)
		} else {
			break
		}
	}
	log.Debug().
		Str("ip", whoami.Status.PodIP).
		Msg("Pod ip detected, carry on...")

	netpongClient, _ = netpong.NewClient(whoami)

	// start periodic test
	go testLoop(clientset, *namespacePrefix)

	// start webserver
	http.HandleFunc("/ping", netpongClient.HandlePing)

	log.Fatal().Err(http.ListenAndServe(*listenAddress, nil)).Msg("ListenAndServe failed")
}

func testLoop(clientset *kubernetes.Clientset, namespacePrefix string) {

	for i := 0; i < 10; i++ {
		log.Debug().Msgf("Waiting %d seconds to start ping test on %s", *pingFrequency, netpongClient.GetWhoamiPod().Status.PodIP)
		time.Sleep(time.Duration(*pingFrequency) * time.Second)

		if targetPod == nil {
			// Scan for changes in pod neighborhood
			k8s.Scan(clientset, namespacePrefix)

			if len(k8s.Pods) > 0 {
				// Select random pod not including this pod and ping it.
				eligiblePods := k8s.PodsExcluding(netpongClient.GetWhoamiPod())
				if len(eligiblePods) == 0 {
					// Pod does not yet have address, wait and try again
					log.Warn().Msg("Not enough eligible pods in namespace, waiting...")
					continue
				}
				p := eligiblePods[rand.Intn(len(eligiblePods))]

				if p.Status.PodIP == "" {
					// Pod does not yet have address, wait and try again
					log.Warn().Msg("Pod did not have valid IP, trying again")
					continue
				}

				log.Debug().
					Str("ip", p.Status.PodIP).
					Msg("Target pod selected")
				targetPod = &p
			} else {
				log.Warn().Msg("No suitable pods in namespace, cannot start ping test")
				continue
			}
		}

		// store reference to target pod so client knows to forward to it in future
		netpongClient.SetTargetPod(targetPod)

		go netpongClient.TestPing(targetPod)

	}
}
