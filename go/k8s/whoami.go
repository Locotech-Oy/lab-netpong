package k8s

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
)

var selfPod *v1.Pod

// Whoami resolves the pod instance in which the program is running.
func Whoami(pods []v1.Pod) (*v1.Pod, error) {

	log.Debug().
		Int("numPods", len(pods)).
		Msg("Num pods to check")

	for _, pod := range pods {
		log.Debug().
			Str("name", pod.GetName()).
			Str("name2", pod.Name).
			Str("ip", pod.Status.PodIP).
			Str("host", pod.Status.HostIP).
			Str("Hostname", os.Getenv("HOSTNAME")).
			Msg("Checking pod against hostname")

		if pod.Name == os.Getenv("HOSTNAME") {
			selfPod = &pod

			log.Debug().
				Str("name", pod.GetName()).
				Str("ip", pod.Status.PodIP).
				Str("host", pod.Status.HostIP).
				Msg("whoami")
			return selfPod, nil
		}
	}

	err := errors.New("Could not resolve whoami pod")

	return nil, err
}
