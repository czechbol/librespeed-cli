package cmd

import (
	"fmt"
	"math"
	"time"

	"github.com/briandowns/spinner"
	"github.com/czechbol/librespeedtest/defs"
	"github.com/czechbol/librespeedtest/speedtest"

	log "github.com/sirupsen/logrus"
)

const (
	pingCount = 10
)

func verboseSpeedTest(cliOpts *CLIOptions) error {
	// Server ranking
	var pb *spinner.Spinner
	var err error
	if cliOpts.Format == "human-readable" {
		pb = spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		pb.Suffix = " Selecting the fastest server based on ping..."
		pb.Start()
	}
	cliOpts.TestServer, err = speedtest.RankServers(&(*cliOpts).ServerList)
	if err != nil {
		return err
	}
	if pb != nil {
		pb.FinalMSG = fmt.Sprintf(
			"Selected server: %s [%s]\n",
			cliOpts.TestServer.Name,
			cliOpts.TestServer.Server,
		)
		pb.Stop()
	}

	ispInfo, err := cliOpts.TestServer.WorkaroundGetIPInfo(cliOpts.DistanceUnit)
	if err != nil {
		log.Errorf("Failed to get IP info: %s", err)
		return err
	}

	// Ping and Jitter test
	pb = spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	pb.Suffix = " Pinging server..."
	pb.Start()

	ping, jitter, err := cliOpts.TestServer.ICMPPingAndJitter(pingCount)
	if err != nil {
		return err
	}

	if pb != nil {
		pb.FinalMSG = fmt.Sprintf(
			"Ping: %.2f ms\tJitter: %.2f ms\n",
			ping,
			jitter,
		)
		pb.Stop()
	}

	// Download test
	var downloadValue float64
	var bytesRead int
	if cliOpts.NoDownload {
		log.Info("Download test is disabled")
	} else {
		downloadValue, bytesRead, err = cliOpts.TestServer.ManualDownload(true, cliOpts.Bytes, cliOpts.BinaryBase, cliOpts.Concurrent, cliOpts.Chunks, time.Duration(cliOpts.Duration)*time.Second)
		if err != nil {
			log.Errorf("Failed to get download speed: %s", err)
			return err
		}
	}

	// Upload test
	var uploadValue float64
	var bytesWritten int
	if cliOpts.NoUpload {
		log.Info("Upload test is disabled")
	} else {
		uploadValue, bytesWritten, err = cliOpts.TestServer.ManualUpload(cliOpts.NoPreAllocate, true, cliOpts.Bytes, cliOpts.BinaryBase, cliOpts.Concurrent, cliOpts.Chunks, time.Duration(cliOpts.Duration)*time.Second)
		if err != nil {
			log.Errorf("Failed to get upload speed: %s", err)
			return err
		}
	}

	report := defs.Report{
		Timestamp:     time.Now(),
		Ping:          math.Round(ping*100) / 100,
		Jitter:        math.Round(jitter*100) / 100,
		Download:      math.Round(downloadValue*100) / 100,
		Upload:        math.Round(uploadValue*100) / 100,
		BytesReceived: bytesRead,
		BytesSent:     bytesWritten,
		Server:        cliOpts.TestServer,
		Client:        defs.Client{IPInfoResponse: ispInfo.RawISPInfo},
	}

	// print share link if --share is given
	if cliOpts.Share {
		var extra defs.TelemetryExtra
		extra.ServerName = cliOpts.TestServer.Name
		extra.Extra = ""
		telemetryServer := defs.TelemetryServer{
			Level:  speedtest.DefaultTelemetryLevel,
			Server: speedtest.DefaultTelemetryServer,
			Path:   speedtest.DefaultTelemetryPath,
			Share:  speedtest.DefaultTelemetryShare,
		}
		if link, err := speedtest.SendTelemetry(telemetryServer, extra, ispInfo, &report, &cliOpts.TestServer.TLog); err != nil {
			log.Errorf("Error when sending telemetry data: %s", err)
		} else {
			report.ShareLink = link
			log.Warnf("Share your result: %s", link)
		}
	}
	return nil
}
