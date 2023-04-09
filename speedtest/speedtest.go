package speedtest

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocarina/gocsv"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/czechbol/librespeed-cli/defs"
	"github.com/czechbol/librespeed-cli/report"
)

const (
	// serverListUrl is the default remote server JSON URL
	serverListUrl = `https://librespeed.org/backend-servers/servers.php`

	defaultTelemetryLevel  = "basic"
	defaultTelemetryServer = "https://librespeed.org"
	defaultTelemetryPath   = "/results/telemetry.php"
	defaultTelemetryShare  = "/results/"
)

type PingJob struct {
	Index  int
	Server defs.Server
}

type PingResult struct {
	Index int
	Ping  float64
}

// SpeedTest is the actual main function that handles the speed test(s)
func RunCli(c *cli.Context) error {
	// check for suppressed output flags
	var silent bool
	if c.Bool(defs.OptionSimple) || c.Bool(defs.OptionJSON) || c.Bool(defs.OptionCSV) {
		log.SetLevel(log.WarnLevel)
		silent = true
	}

	// check for debug flag
	if c.Bool(defs.OptionDebug) {
		log.SetLevel(log.DebugLevel)
	}

	// print help
	if c.Bool(defs.OptionHelp) {
		return cli.ShowAppHelp(c)
	}

	// print version
	if c.Bool(defs.OptionVersion) {
		log.SetOutput(os.Stdout)
		log.Warnf("%s %s (built on %s)", defs.ProgName, defs.ProgVersion, defs.BuildDate)
		log.Warn("https://github.com/czechbol/librespeed-cli")
		log.Warn("Licensed under GNU Lesser General Public License v3.0")
		log.Warn("LibreSpeed\tCopyright (C) 2016-2020 Federico Dossena")
		log.Warn("librespeed-cli\tCopyright (C) 2020 Maddie Zhan")
		log.Warn("librespeed.org\tCopyright (C)")
		return nil
	}

	// set CSV delimiter
	gocsv.TagSeparator = c.String(defs.OptionCSVDelimiter)

	// if --csv-header is given, print the header and exit (same behavior speedtest-cli)
	if c.Bool(defs.OptionCSVHeader) {
		var rep []report.FlatReport
		header, _ := gocsv.MarshalString(&rep)
		fmt.Print(header)
		return nil
	}

	testOpts, err := parseCliOptions(c)
	if err != nil {
		log.Errorf("Error parsing CLI flags: %s", err)
		return err
	}

	// HTTP requests timeout
	http.DefaultClient.Timeout = time.Duration(testOpts.Timeout) * time.Second

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: c.Bool(defs.OptionSkipCertVerify)}

	// bind to source IP address if given, or if ipv4/ipv6 is forced
	if src := c.String(defs.OptionSource); src != "" || (testOpts.IPv4() || testOpts.IPv6()) {
		var localTCPAddr *net.TCPAddr
		if src != "" {
			// first we parse the IP to see if it's valid
			addr, err := net.ResolveIPAddr(testOpts.Network, src)
			if err != nil {
				if strings.Contains(err.Error(), "no suitable address") {
					if testOpts.IPv6() {
						log.Errorf("Address %s is not a valid IPv6 address", src)
					} else {
						log.Errorf("Address %s is not a valid IPv4 address", src)
					}
				} else {
					log.Errorf("Error parsing source IP: %s", err)
				}
				return err
			}

			log.Debugf("Using %s as source IP", src)
			localTCPAddr = &net.TCPAddr{IP: addr.IP}
		}

		var dialContext func(context.Context, string, string) (net.Conn, error)
		defaultDialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}

		if localTCPAddr != nil {
			defaultDialer.LocalAddr = localTCPAddr
		}

		switch {
		case testOpts.IPv4():
			dialContext = func(ctx context.Context, network, address string) (conn net.Conn, err error) {
				return defaultDialer.DialContext(ctx, "tcp4", address)
			}
		case testOpts.IPv6():
			dialContext = func(ctx context.Context, network, address string) (conn net.Conn, err error) {
				return defaultDialer.DialContext(ctx, "tcp6", address)
			}
		default:
			dialContext = defaultDialer.DialContext
		}

		// set default HTTP client's Transport to the one that binds the source address
		// this is modified from http.DefaultTransport
		transport.DialContext = dialContext
	}

	http.DefaultClient.Transport = transport

	// if --list is given, print servers and exit
	if c.Bool(defs.OptionList) {
		for _, svr := range testOpts.ServerList {
			var sponsorMsg string
			if svr.Sponsor() != "" {
				sponsorMsg = fmt.Sprintf(" [Sponsor: %s]", svr.Sponsor())
			}
			log.Warnf("%d: %s (%s) %s", svr.ID, svr.Name, svr.Server, sponsorMsg)
		}
		return nil
	}

	return CliSpeedTest(testOpts, c, silent)
}

// Parsing command line flags and returning resulting defs.TestOptions
func parseCliOptions(c *cli.Context) (*defs.TestOptions, error) {
	forceIPv4 := c.Bool(defs.OptionIPv4)
	forceIPv6 := c.Bool(defs.OptionIPv6)

	var network string
	switch {
	case forceIPv4:
		network = "ip4"
	case forceIPv6:
		network = "ip6"
	default:
		network = "ip"
	}

	servers, err := getServerList(c)
	if err != nil {
		log.Errorf("Error when fetching server list: %s", err)
		return nil, err
	}

	if len(c.IntSlice(defs.OptionServer)) > 0 || len(c.IntSlice(defs.OptionExclude)) > 0 {
		specifyServers(servers, c.IntSlice(defs.OptionServer))
		excludeServers(servers, c.IntSlice(defs.OptionExclude))
	} else {
		// else select the fastest server from the list
		log.Info("Selecting the fastest server based on ping")
		server, err := RankServers(servers, c.String(defs.OptionSource), network, c.Bool(defs.OptionNoICMP))
		if err != nil {
			log.Errorf("Error when ranking servers: %s", err)
			return nil, err
		}
		servers = &[]defs.Server{*server}
	}

	if err != nil {
		log.Errorf("Error when selecting servers: %s", err)
		return nil, err
	}

	telemetryOpts, err := parseTelemetryOptions(c)
	if err != nil {
		log.Errorf("Error when fetching server list: %s", err)
		return nil, err
	}
	var concurrentNum int
	if concurrentNum = c.Int(defs.OptionConcurrent); concurrentNum <= 0 {
		log.Errorf("Concurrent requests cannot be lower than 1: %d is given", concurrentNum)
		return nil, errors.New("invalid concurrent requests setting")
	}

	testOpts := defs.TestOptions{
		Chunks:          c.Int(defs.OptionChunks),
		Concurrent:      concurrentNum,
		BinaryBase:      c.Bool(defs.OptionBinaryBase),
		Bytes:           c.Bool(defs.OptionBytes),
		DistanceUnit:    defs.DistanceUnit(c.String(defs.OptionDistance)),
		Duration:        c.Int(defs.OptionDuration),
		Network:         network,
		NoDownload:      c.Bool(defs.OptionNoDownload),
		NoICMP:          c.Bool(defs.OptionNoICMP),
		NoPreAllocate:   c.Bool(defs.OptionNoPreAllocate),
		NoUpload:        c.Bool(defs.OptionNoUpload),
		Secure:          c.Bool(defs.OptionSecure),
		ServerList:      *servers,
		Share:           c.Bool(defs.OptionShare),
		SkipCertVerify:  c.Bool(defs.OptionSkipCertVerify),
		SourceIP:        c.String(defs.OptionSource),
		TelemetryServer: *telemetryOpts,
		TelemetryExtra:  c.String(defs.OptionTelemetryExtra),
		Timeout:         c.Int(defs.OptionTimeout),
		UploadSize:      c.Int(defs.OptionUploadSize),
	}

	return &testOpts, nil
}

func parseTelemetryOptions(c *cli.Context) (*defs.TelemetryServer, error) {
	// read telemetry settings if --share or any --telemetry option is given
	var telemetryServer defs.TelemetryServer
	telemetryJSON := c.String(defs.OptionTelemetryJSON)
	telemetryLevel := c.String(defs.OptionTelemetryLevel)
	telemetryServerString := c.String(defs.OptionTelemetryServer)
	telemetryPath := c.String(defs.OptionTelemetryPath)
	telemetryShare := c.String(defs.OptionTelemetryShare)
	if c.Bool(defs.OptionShare) || telemetryJSON != "" || telemetryLevel != "" || telemetryServerString != "" || telemetryPath != "" || telemetryShare != "" {
		if telemetryJSON != "" {
			b, err := ioutil.ReadFile(telemetryJSON)
			if err != nil {
				log.Errorf("Cannot read %s: %s", telemetryJSON, err)
				return nil, err
			}
			if err := json.Unmarshal(b, &telemetryServer); err != nil {
				log.Errorf("Error parsing %s: %s", err)
				return nil, err
			}
		}

		if telemetryLevel != "" {
			if telemetryLevel != "disabled" && telemetryLevel != "basic" && telemetryLevel != "full" && telemetryLevel != "debug" {
				log.Fatalf("Unsupported telemetry level: %s", telemetryLevel)
			}
			telemetryServer.Level = telemetryLevel
		} else if telemetryServer.Level == "" {
			telemetryServer.Level = defaultTelemetryLevel
		}

		if telemetryServerString != "" {
			telemetryServer.Server = telemetryServerString
		} else if telemetryServer.Server == "" {
			telemetryServer.Server = defaultTelemetryServer
		}

		if telemetryPath != "" {
			telemetryServer.Path = telemetryPath
		} else if telemetryServer.Path == "" {
			telemetryServer.Path = defaultTelemetryPath
		}

		if telemetryShare != "" {
			telemetryServer.Share = telemetryShare
		} else if telemetryServer.Share == "" {
			telemetryServer.Share = defaultTelemetryShare
		}
	}
	return &telemetryServer, nil
}

func getServerList(c *cli.Context) (*[]defs.Server, error) {
	// load server list
	var servers []defs.Server
	var err error
	if str := c.String(defs.OptionLocalJSON); str != "" {
		switch str {
		case "-":
			// load server list from stdin
			log.Info("Using local JSON server list from stdin")
			servers, err = getLocalServersReader(c.Bool(defs.OptionSecure), os.Stdin, c.Bool(defs.OptionNoICMP))
		default:
			// load server list from local JSON file
			log.Infof("Using local JSON server list: %s", str)
			servers, err = getLocalServers(c.Bool(defs.OptionSecure), str, c.Bool(defs.OptionNoICMP))
		}
	} else {
		// fetch the server list JSON and parse it into the `servers` array
		serverUrl := serverListUrl
		if str := c.String(defs.OptionServerJSON); str != "" {
			serverUrl = str
		}
		log.Infof("Retrieving server list from %s", serverUrl)

		servers, err = FetchServerList(c.Bool(defs.OptionSecure), serverUrl, c.Bool(defs.OptionNoICMP))

		if err != nil {
			log.Info("Retry with /.well-known/librespeed")
			servers, err = FetchServerList(c.Bool(defs.OptionSecure), serverUrl+"/.well-known/librespeed", c.Bool(defs.OptionNoICMP))
		}
	}
	if err != nil {
		log.Errorf("Error when fetching server list: %s", err)
		return nil, err
	}

	return &servers, nil
}

// Removes the excluded server ids from the Server list
func specifyServers(servers *[]defs.Server, specifiedIds []int) *[]defs.Server {
	if len(specifiedIds) == 0 {
		return servers
	}

	m := make(map[int]bool)

	for _, item := range specifiedIds {
		m[item] = true
	}

	var tempList []defs.Server

	for _, item := range *servers {
		if _, ok := m[item.ID]; ok {
			tempList = append(tempList, item)
		}
	}
	return &tempList
}

// Removes the excluded server ids from the Server list
func excludeServers(servers *[]defs.Server, excludeIds []int) *[]defs.Server {
	if len(excludeIds) == 0 {
		return servers
	}
	m := make(map[int]bool)

	for _, item := range excludeIds {
		m[item] = true
	}

	var tempList []defs.Server

	for _, item := range *servers {
		if _, ok := m[item.ID]; !ok {
			tempList = append(tempList, item)
		}
	}
	return &tempList
}

// Ranking Librespeed servers and selecting the fastest one based on ping
func RankServers(servers *[]defs.Server, sourceIp string, network string, noIcmp bool) (*defs.Server, error) {
	var wg sync.WaitGroup
	jobs := make(chan PingJob, len(*servers))
	results := make(chan PingResult, len(*servers))
	done := make(chan struct{})

	pingList := make(map[int]float64)

	// spawn 10 concurrent pingers
	for i := 0; i < 10; i++ {
		go pingWorker(jobs, results, &wg, sourceIp, network, noIcmp)
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
		return nil, errors.New("No server is currently available, please try again later.")
	}

	// get the fastest server's index in the `servers` array
	var serverIdx int
	for idx, ping := range pingList {
		if ping > 0 && ping <= pingList[serverIdx] {
			serverIdx = idx
		}
	}
	return &(*servers)[serverIdx], nil
}

func pingWorker(jobs <-chan PingJob, results chan<- PingResult, wg *sync.WaitGroup, srcIp, network string, noICMP bool) {
	for {
		job := <-jobs
		server := job.Server
		// get the URL of the speed test server from the JSON
		u, err := server.GetURL()
		if err != nil {
			log.Debugf("Server URL is invalid for %s (%s), skipping", server.Name, server.Server)
			wg.Done()
			return
		}

		// check the server is up by accessing the ping URL and checking its returned value == empty and status code == 200
		if server.IsUp() {
			// skip ICMP if option given
			server.NoICMP = noICMP

			// if server is up, get ping
			ping, _, err := server.ICMPPingAndJitter(1, srcIp, network)
			if err != nil {
				log.Debugf("Can't ping server %s (%s), skipping", server.Name, u.Hostname())
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

// getServerList fetches the server JSON from a remote server
func FetchServerList(forceHTTPS bool, serverList string, noICMP bool) ([]defs.Server, error) {
	// getting the server list from remote
	var servers []defs.Server
	req, err := http.NewRequest(http.MethodGet, serverList, nil)
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

	return preprocessServers(servers, forceHTTPS, noICMP)
}

// getLocalServersReader loads the server JSON from an io.Reader
func getLocalServersReader(forceHTTPS bool, jsonFile *os.File, noICMP bool) ([]defs.Server, error) {
	defer jsonFile.Close()

	var servers []defs.Server

	b, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &servers); err != nil {
		return nil, err
	}

	return preprocessServers(servers, forceHTTPS, noICMP)
}

// getLocalServers loads the server JSON from a local file
func getLocalServers(forceHTTPS bool, jsonFilePath string, noICMP bool) ([]defs.Server, error) {
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	return getLocalServersReader(forceHTTPS, jsonFile, noICMP)
}

// PreprocessServers makes some needed modifications to the servers fetched
func preprocessServers(servers []defs.Server, forceHTTPS bool, noICMP bool) ([]defs.Server, error) {
	for i := range servers {
		u, err := servers[i].GetURL()
		if err != nil {
			return nil, err
		}

		// if no scheme is defined, use http as default, or https when --secure is given in cli options
		// if the scheme is predefined and --secure is not given, we will use it as-is
		if forceHTTPS {
			u.Scheme = "https"
		} else if u.Scheme == "" {
			// if `secure` is not used and no scheme is defined, use http
			u.Scheme = "http"
		}

		servers[i].NoICMP = noICMP

		// modify the server struct in the array in place
		servers[i].Server = u.String()
	}

	return servers, nil
}
