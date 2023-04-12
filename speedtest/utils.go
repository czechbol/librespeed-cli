package speedtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/czechbol/librespeedtest/defs"
	log "github.com/sirupsen/logrus"
)

const (
	// serverListUrl is the default remote server JSON URL
	ServerListUrl = `https://librespeed.org/backend-servers/servers.php`

	DefaultPingCount       = 10
	DefaultTelemetryLevel  = "basic"
	DefaultTelemetryServer = "https://librespeed.org"
	DefaultTelemetryPath   = "/results/telemetry.php"
	DefaultTelemetryShare  = "/results/"
)

type PingJob struct {
	Index  int
	Server defs.Server
}

type PingResult struct {
	Index int
	Ping  float64
}

// FetchServerList fetches a server list from a URL
func FetchServerList(listURL string) (*[]defs.Server, error) {
	// getting the server list from remote
	var servers []defs.Server
	req, err := http.NewRequest(http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defs.UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.Unmarshal(b, &servers); err != nil {
		return nil, err
	}
	return &servers, nil
}

// GetLocalServerList reads a server list from a filesystem path
func GetLocalServerList(listPath string) (*[]defs.Server, error) {
	f, err := os.OpenFile(listPath, os.O_RDONLY, 0o644)
	if err != nil {
		fmt.Println("Error opening file")
		return nil, err
	}

	defer f.Close()

	var servers []defs.Server

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &servers); err != nil {
		return nil, err
	}

	return &servers, err
}

// PreprocessServers sets a few key attributes of the servers
func PreprocessServers(
	servers *[]defs.Server,
	forceHTTPS bool,
	noICMP bool,
) error {
	for i := range *servers {
		u, err := (*servers)[i].GetURL()
		if err != nil {
			return err
		}

		// if no scheme is defined, use http as default, or https when --secure is given in cli options
		// if the scheme is predefined and --secure is not given, we will use it as-is
		if forceHTTPS {
			u.Scheme = "https"
		} else if u.Scheme == "" {
			// if `secure` is not used and no scheme is defined, use http
			u.Scheme = "http"
		}

		(*servers)[i].NoICMP = noICMP

		// modify the server struct in the array in place
		(*servers)[i].Server = u.String()
	}

	return nil
}

// RankServer performs a ping request to each server frin the given slice and
// returns the fastest one
func RankServers(servers *[]defs.Server) (defs.Server, error) {
	var wg sync.WaitGroup
	jobs := make(chan PingJob, len(*servers))
	results := make(chan PingResult, len(*servers))
	done := make(chan struct{})

	pingList := make(map[int]float64)

	// spawn concurrent pingers
	for i := 0; i < len(*servers); i++ {
		go pingWorker(jobs, results, &wg)
	}
	// send ping jobs to workers
	for idx, server := range *servers {
		wg.Add(1)
		jobs <- PingJob{Index: idx, Server: server}
	}

	go func() {
		wg.Wait()
		close(done)
	}()

Loop:
	for {
		select {
		case result := <-results:
			pingList[result.Index] = result.Ping
		case <-done:
			break Loop
		}
	}

	if len(pingList) == 0 {
		return defs.Server{}, errors.New(
			"No server is currently available, please try again later.",
		)
	}

	// get the fastest server's index in the `servers` array
	var serverIdx int
	for idx, ping := range pingList {
		if ping > 0 && ping <= pingList[serverIdx] {
			serverIdx = idx
		}
	}
	return (*servers)[serverIdx], nil
}

func pingWorker(
	jobs <-chan PingJob,
	results chan<- PingResult,
	wg *sync.WaitGroup,
) {
	for {
		job := <-jobs
		server := job.Server
		// get the URL of the speed test server from the JSON
		u, err := server.GetURL()
		if err != nil {
			log.Debugf(
				"Server URL is invalid for %s (%s), skipping",
				server.Name,
				server.Server,
			)
			wg.Done()
			return
		}

		// check the server is up by accessing the ping URL and checking its returned value == empty and status code == 200
		if server.IsUp() {

			// if server is up, get ping
			ping, _, err := server.ICMPPingAndJitter(1)
			if err != nil {
				log.Debugf(
					"Can't ping server %s (%s), skipping",
					server.Name,
					u.Hostname(),
				)
				wg.Done()
				return
			}
			// return result
			results <- PingResult{Index: job.Index, Ping: ping}
			wg.Done()
		} else {
			log.Debugf("Server %s (%s) doesn't seem to be up, skipping", server.Name, u.Hostname())
			wg.Done()
		}
	}
}
