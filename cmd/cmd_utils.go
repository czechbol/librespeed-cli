package cmd

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/czechbol/librespeedtest/defs"
)

const (
	// serverListUrl is the default remote server JSON URL
	serverListUrl = `https://librespeed.org/backend-servers/servers.php`

	defaultTelemetryLevel  = "basic"
	defaultTelemetryServer = "https://librespeed.org"
	defaultTelemetryPath   = "/results/telemetry.php"
	defaultTelemetryShare  = "/results/"
)

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

func GetLocalServerList(listPath string) (*[]defs.Server, error) {
	f, err := os.OpenFile(listPath, os.O_RDONLY, 0644)
	if err != nil {
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
