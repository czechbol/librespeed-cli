package speedtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/czechbol/librespeedtest/defs"
	log "github.com/sirupsen/logrus"
)

// AutoSpeedTest is a function that selects the fastest server
// and runs a fully automatic speedtest with default parameters,
// distanceUnit shoulr be one of ["mi", "km", "NM"] (miles, kilometers, nautical miles)
func AutoSpeedTest(
	distanceUnit string,
	forceHTTPS bool,
	noICMP bool,
	noShare bool,
) (*defs.Report, error) {
	var serverList *[]defs.Server
	var testServer defs.Server
	var err error
	if serverList, err = FetchServerList(ServerListUrl); err != nil {
		return nil, err
	}
	if err = PreprocessServers(serverList, forceHTTPS, noICMP); err != nil {
		return nil, err
	}
	if testServer, err = RankServers(serverList); err != nil {
		return nil, err
	}
	return SingleSpeedTest(
		&testServer,
		false,
		false,
		DefaultPingCount,
		distanceUnit,
		3,
		100,
		false,
		1024,
		time.Duration(15)*time.Second,
		noShare,
	)
}

// SingleSpeedTest runs a speedtest for one server and returns a corresponding Report object
// distanceUnit shoulr be one of ["mi", "km", "NM"] (miles, kilometers, nautical miles)
func SingleSpeedTest(
	server *defs.Server,
	noDownload bool,
	noUpload bool,
	pingCount int,
	distanceUnit string,
	requests int,
	chunks int,
	noPrealloc bool,
	uploadSize int,
	duration time.Duration,
	noShare bool,
) (*defs.Report, error) {
	report := defs.Report{Server: *server}

	log.Info("Getting ISP information")
	ispInfo, err := server.WorkaroundGetIPInfo(distanceUnit)
	if err != nil {
		log.Errorf("Failed to get IP info: %s", err)
		return nil, err
	}
	report.Client = defs.Client{IPInfoResponse: ispInfo.RawISPInfo}

	log.Info("Ping and Jitter test started")
	if report.Ping, report.Jitter, err = server.ICMPPingAndJitter(pingCount); err != nil {
		return nil, err
	}
	if !noDownload {
		log.Info("Download test started")
		if report.Download, report.BytesReceived, err = server.Download(requests, chunks, duration); err != nil {
			return nil, err
		}
	}
	if !noUpload {
		log.Info("Upload tests started")
		if report.Upload, report.BytesSent, err = server.Upload(noPrealloc, requests, uploadSize, duration); err != nil {
			return nil, err
		}
	}
	report.Timestamp = time.Now()

	if !noShare {
		var extra defs.TelemetryExtra
		extra.ServerName = server.Name
		extra.Extra = ""
		telemetryServer := defs.TelemetryServer{
			Level:  DefaultTelemetryLevel,
			Server: DefaultTelemetryServer,
			Path:   DefaultTelemetryPath,
			Share:  DefaultTelemetryShare,
		}
		log.Info("Sending telemetry information")
		if link, err := SendTelemetry(telemetryServer, extra, ispInfo, &report, &server.TLog); err != nil {
			log.Errorf("Error when sending telemetry data: %s", err)
		} else {
			report.ShareLink = link
		}
	}

	return &report, nil
}

// sendTelemetry sends the telemetry result to server, if --share is given
func SendTelemetry(
	telemetryServer defs.TelemetryServer,
	extra defs.TelemetryExtra,
	ispInfo *defs.GetIPResult,
	report *defs.Report,
	telemetryLog *defs.TelemetryLog,
) (string, error) {
	var buf bytes.Buffer
	wr := multipart.NewWriter(&buf)

	b, _ := json.Marshal(*ispInfo)
	if fIspInfo, err := wr.CreateFormField("ispinfo"); err != nil {
		log.Debugf("Error creating form field: %s", err)
		return "", err
	} else if _, err = fIspInfo.Write(b); err != nil {
		log.Debugf("Error writing form field: %s", err)
		return "", err
	}

	if fDownload, err := wr.CreateFormField("dl"); err != nil {
		log.Debugf("Error creating form field: %s", err)
		return "", err
	} else if _, err = fDownload.Write([]byte(strconv.FormatFloat(report.Download, 'f', 2, 64))); err != nil {
		log.Debugf("Error writing form field: %s", err)
		return "", err
	}

	if fUpload, err := wr.CreateFormField("ul"); err != nil {
		log.Debugf("Error creating form field: %s", err)
		return "", err
	} else if _, err = fUpload.Write([]byte(strconv.FormatFloat(report.Upload, 'f', 2, 64))); err != nil {
		log.Debugf("Error writing form field: %s", err)
		return "", err
	}

	if fPing, err := wr.CreateFormField("ping"); err != nil {
		log.Debugf("Error creating form field: %s", err)
		return "", err
	} else if _, err = fPing.Write([]byte(strconv.FormatFloat(report.Ping, 'f', 2, 64))); err != nil {
		log.Debugf("Error writing form field: %s", err)
		return "", err
	}

	if fJitter, err := wr.CreateFormField("jitter"); err != nil {
		log.Debugf("Error creating form field: %s", err)
		return "", err
	} else if _, err = fJitter.Write([]byte(strconv.FormatFloat(report.Jitter, 'f', 2, 64))); err != nil {
		log.Debugf("Error writing form field: %s", err)
		return "", err
	}

	if fLog, err := wr.CreateFormField("log"); err != nil {
		log.Debugf("Error creating form field: %s", err)
		return "", err
	} else if _, err = fLog.Write([]byte(telemetryLog.String())); err != nil {
		log.Debugf("Error writing form field: %s", err)
		return "", err
	}

	b, _ = json.Marshal(extra)
	if fExtra, err := wr.CreateFormField("extra"); err != nil {
		log.Debugf("Error creating form field: %s", err)
		return "", err
	} else if _, err = fExtra.Write(b); err != nil {
		log.Debugf("Error writing form field: %s", err)
		return "", err
	}

	if err := wr.Close(); err != nil {
		log.Debugf("Error flushing form field writer: %s", err)
		return "", err
	}

	telemetryUrl, err := telemetryServer.GetPath()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, telemetryUrl.String(), &buf)
	if err != nil {
		log.Debugf("Error when creating HTTP request: %s", err)
		return "", err
	}
	req.Header.Set("Content-Type", wr.FormDataContentType())
	req.Header.Set("User-Agent", defs.UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Debugf("Error when making HTTP request: %s", err)
		return "", err
	}
	fmt.Println(resp)
	defer resp.Body.Close()

	id, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error when reading HTTP request: %s", err)
		return "", err
	}

	resultUrl, err := telemetryServer.GetShare()
	if err != nil {
		return "", err
	}
	fmt.Println(string(id))

	if str := strings.Split(string(id), " "); len(str) != 2 {
		return "", fmt.Errorf("server returned invalid response: %s", id)
	} else {
		q := resultUrl.Query()
		q.Set("id", str[1])
		resultUrl.RawQuery = q.Encode()

		return resultUrl.String(), nil
	}
}
