package netpong

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
)

type Pingload struct {
	Uuid     string              `json:"uuid"`
	Numhops  int64               `json:"numhops"`
	Hostlist map[string]Pinghost `json:"hostlist"`
}

type Pinghost struct {
	Address string `json:"address"`
	Podname string `json:"podname"`
}

type NetpongClient struct {
	whoamiPod *v1.Pod
	targetPod *v1.Pod
}

func NewClient(whoamiPod *v1.Pod) (*NetpongClient, error) {
	c := &NetpongClient{
		whoamiPod: whoamiPod,
	}

	log.Debug().
		Interface("whoami", &whoamiPod).
		Msg("Creating new client")

	return c, nil
}

func (c *NetpongClient) SetTargetPod(pod *v1.Pod) {
	c.targetPod = pod
}

func (c *NetpongClient) GetWhoamiPod() (pod *v1.Pod) {
	return c.whoamiPod
}

func (c *NetpongClient) TestPing(pod *v1.Pod) {
	url := fmt.Sprintf("http://%s:8080/ping", pod.Status.PodIP)

	_, err := c.testHTTP(url)
	if err != nil {
		log.
			Warn().
			Err(err).
			Str("url", url).
			Msg("Test ping failed")
	}
}

func (c *NetpongClient) TestPingWithPayload(pod *v1.Pod, payload Pingload) {
	// delay forwarding
	time.Sleep(time.Duration(5) * time.Second)
	url := fmt.Sprintf("http://%s:8080/ping", pod.Status.PodIP)

	_, err := c.testHTTPWithPayload(url, payload)
	if err != nil {
		log.
			Warn().
			Err(err).
			Str("url", url).
			Msg("Test ping failed")
	}
}

func (c *NetpongClient) testHTTP(url string) (time.Time, error) {

	// Create a Version 4 UUID.
	u2, err := uuid.NewV4()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate UUID")
	}

	body := Pingload{
		Uuid:     u2.String(),
		Numhops:  0,
		Hostlist: map[string]Pinghost{},
	}

	return c.testHTTPWithPayload(url, body)

}

func (c *NetpongClient) testHTTPWithPayload(url string, payload Pingload) (time.Time, error) {

	jsonValue, _ := json.Marshal(payload)

	log.Debug().
		Str("url", url).
		Str("uuid", payload.Uuid).
		RawJSON("body", jsonValue).
		Msgf("Sending ping to %s", url)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating HTTP request")
	}

	req.Header.Add("Content-Type", "application/json")

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

func (c *NetpongClient) HandlePing(w http.ResponseWriter, r *http.Request) {

	var p Pingload
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	log.Debug().Interface("body", p).Msg("Received ping request")

	// increment the counter on the payload, append this host to the host list and forward package to next pod
	p.Numhops++

	if _, exists := p.Hostlist[c.whoamiPod.Status.PodIP]; !exists {
		p.Hostlist[c.whoamiPod.Status.PodIP] = Pinghost{
			Address: c.whoamiPod.Status.PodIP,
			Podname: c.whoamiPod.Name,
		}
	}

	if c.targetPod != nil && p.Numhops < 10 {
		log.Debug().Msg("Forwarding ping request")
		go c.TestPingWithPayload(c.targetPod, p)
	}

	// Send a response to the original pod
	w.Write([]byte("pong"))
}
