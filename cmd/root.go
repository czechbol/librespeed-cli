/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/czechbol/librespeedtest/defs"
	"github.com/czechbol/librespeedtest/speedtest"
	"github.com/gocarina/gocsv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	cmdUse   = "librespeedtest"
	cmdShort = "librespeedtest - Test your Internet speed with LibreSpeed 🚀"
	cmdLong  = `librespeedtest - Test your Internet speed with LibreSpeed 🚀

No Flash, No Java, No Websocket, No Bullshit. 
Librespeed is an open-source speedtest for measuring your Internet speed.`
)

var (
	listServers bool
	csvHeader   bool
	formatCheck = map[string]bool{
		"human-readable": true,
		"simple":         true,
		"csv":            true,
		"tsv":            true,
		"json":           true,
		"jsonl":          true,
		"json-pretty":    true,
	}
	distanceCheck = map[string]bool{
		"km": true,
		"mi": true,
		"NM": true,
	}
)

type CLIOptions struct {
	Chunks          int                  `json:"chunks"`
	Concurrent      int                  `json:"concurrent"`
	BinaryBase      bool                 `json:"binary_base,omitempty"`
	Bytes           bool                 `json:"bytes,omitempty"`
	DistanceUnit    string               `json:"distance_unit"`
	Duration        int                  `json:"duration"`
	NoDownload      bool                 `json:"no_download,omitempty"`
	NoICMP          bool                 `json:"no_icmp,omitempty"`
	NoPreAllocate   bool                 `json:"no_pre_allocate,omitempty"`
	NoUpload        bool                 `json:"no_upload,omitempty"`
	Secure          bool                 `json:"secure,omitempty"`
	TestServer      defs.Server          `json:"test_server,omitempty"`
	Share           bool                 `json:"share,omitempty"`
	SkipCertVerify  bool                 `json:"skip_cert_verify,omitempty"`
	SourceIP        string               `json:"source_ip,omitempty"`
	TelemetryServer defs.TelemetryServer `json:"telemetry_server"`
	TelemetryExtra  string               `json:"telemetry_extra,omitempty"`
	UploadSize      int                  `json:"upload_size"`
	Format          string               `json:"format"`
	ForceHTTPS      bool                 `json:"force_https,omitempty"`
	ServerList      []defs.Server        `json:"server_list,omitempty"`
	LogVerbosity    int                  `json:"-"`
}

func (cliOpts *CLIOptions) Complete(args []string) error {
	if !formatCheck[cliOpts.Format] {
		keys := make([]string, 0, len(formatCheck))
		for k := range formatCheck {
			keys = append(keys, k)
		}
		printKeys := "['" + strings.Join(keys, `','`) + `']`
		log.WithFields(log.Fields{
			"got":     cliOpts.Format,
			"allowed": printKeys,
		}).Fatal("Invalid Argument")
	}
	if !distanceCheck[cliOpts.DistanceUnit] {
		keys := make([]string, 0, len(distanceCheck))
		for k := range distanceCheck {
			keys = append(keys, k)
		}
		printKeys := "['" + strings.Join(keys, `','`) + `']`
		log.WithFields(log.Fields{
			"got":     cliOpts.DistanceUnit,
			"allowed": printKeys,
		}).Fatal("Invalid Argument")
	}
	return nil
}

func (cliOpts *CLIOptions) Run(
	cmd *cobra.Command,
	out io.Writer,
) error {
	log.SetLevel(log.Level(3 + cliOpts.LogVerbosity))

	// Print CSV header and exit
	header, err := cmd.Flags().GetBool("csv-header")
	if err != nil {
		return err
	} else if header {
		var rep []defs.FlatReport
		str, _ := gocsv.MarshalString(&rep)
		fmt.Fprint(out, str)
		return nil
	}
	// Print TSV header and exit
	header, err = cmd.Flags().GetBool("tsv-header")
	if err != nil {
		return err
	} else if header {
		gocsv.SetCSVWriter(func(out io.Writer) *gocsv.SafeCSVWriter {
			writer := csv.NewWriter(out)
			writer.Comma = '\t'
			return gocsv.NewSafeCSVWriter(writer)
		})

		var rep []defs.FlatReport
		str, _ := gocsv.MarshalString(&rep)
		fmt.Fprint(out, str)
		return nil
	}

	// Fetch server list
	log.Info("Fetching server list")
	var defaultServerList *[]defs.Server
	if defaultServerList, err = speedtest.FetchServerList(speedtest.ServerListUrl); err != nil {
		log.WithField("url", speedtest.ServerListUrl).
			Error("Unable to fetch remote server list")
		return err
	} else {
		cliOpts.ServerList = append(cliOpts.ServerList, (*defaultServerList)...)
		if err = speedtest.PreprocessServers(&cliOpts.ServerList, cliOpts.ForceHTTPS, cliOpts.NoICMP); err != nil {
			log.Error("Unable to preprocess server list")
			return err
		}
	}

	// Print Server List and exit
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return err
	} else if list {
		for _, server := range cliOpts.ServerList {
			fmt.Fprintln(out, server)
		}
		return nil
	}

	// using verbose output for humans
	if cliOpts.Format == "human-readable" {
		if err = verboseSpeedTest(cliOpts); err != nil {
			return err
		}
		return nil
	}

	log.Info("Selecting the fastest server based on ping")
	var testServer defs.Server
	if testServer, err = speedtest.RankServers(&cliOpts.ServerList); err != nil {
		return err
	}
	log.Info("Starting the speed test")
	report, err := speedtest.SingleSpeedTest(
		&testServer,
		cliOpts.NoDownload,
		cliOpts.NoUpload,
		speedtest.DefaultPingCount,
		cliOpts.DistanceUnit,
		cliOpts.Concurrent,
		cliOpts.Chunks,
		cliOpts.NoPreAllocate,
		cliOpts.UploadSize,
		time.Duration(cliOpts.Duration)*time.Second,
		!cliOpts.Share,
	)

	if cliOpts.Format == "simple" {
		fmt.Printf(`Ping:   %.2f ms Jitter: %.2f ms
Download rate:  %.2f Mbps
Upload rate:    %.2f Mbps
`, report.Ping, report.Jitter, report.Download, report.Upload)
	} else if cliOpts.Format == "csv" {
		flatReport := report.GetFlatReport()
		reportSlice := make([]defs.FlatReport, 1)
		reportSlice[0] = flatReport
		if resultStrig, err := gocsv.MarshalStringWithoutHeaders(&reportSlice); err != nil {
			log.Errorf("Error generating CSV report: %s", err)
		} else {
			fmt.Print(resultStrig)
		}

	} else if cliOpts.Format == "tsv" {
		gocsv.SetCSVWriter(func(out io.Writer) *gocsv.SafeCSVWriter {
			writer := csv.NewWriter(out)
			writer.Comma = '\t'
			return gocsv.NewSafeCSVWriter(writer)
		})
		flatReport := report.GetFlatReport()
		reportSlice := make([]defs.FlatReport, 1)
		reportSlice[0] = flatReport
		if resultStrig, err := gocsv.MarshalStringWithoutHeaders(&reportSlice); err != nil {
			log.Errorf("Error generating CSV report: %s", err)
		} else {
			fmt.Print(resultStrig)
		}

	} else if cliOpts.Format == "json" {
		reportSlice := make([]defs.Report, 1)
		reportSlice[0] = *report
		if jsonBytes, err := json.Marshal(&reportSlice); err != nil {
			log.Errorf("Error generating JSON report: %s", err)
		} else {
			fmt.Println(string(jsonBytes))
		}

	} else if cliOpts.Format == "jsonl" {
		reportSlice := make([]defs.Report, 2)
		reportSlice[0] = *report
		var output string
		for _, rep := range reportSlice {
			if jsonBytes, err := json.Marshal(&rep); err != nil {
				log.Errorf("Error generating JSON report: %s", err)
			} else {
				output = fmt.Sprintf("%s\n%s", output, string(jsonBytes))
			}
		}
		fmt.Println(output)

	} else if cliOpts.Format == "json-pretty" {
		reportSlice := make([]defs.Report, 1)
		reportSlice[0] = *report
		if jsonBytes, err := json.MarshalIndent(&reportSlice, "", "  "); err != nil {
			log.Errorf("Error generating JSON report: %s", err)
		} else {
			fmt.Println(string(jsonBytes))
		}

	}

	return nil
}

func (cliOpts *CLIOptions) CobraCommand() *cobra.Command {
	version := fmt.Sprintf(`%s %s (built on %s)
Licensed under GNU Lesser General Public License v3.0
LibreSpeed	Copyright (C) 2016-2020 Federico Dossena
librespeed-cli	Copyright (C) 2020 Maddie Zhan
librespeedtest	Copyright (C) 2023 czechbol
librespeed.org	Copyright (C)`, defs.ProgName, defs.ProgVersion, defs.BuildDate)

	cmd := &cobra.Command{
		Use:           cmdUse,
		Short:         cmdShort,
		Long:          cmdLong,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	f := cmd.Flags()

	f.BoolP("list", "l", false, "Display a list of LibreSpeed.org servers")
	f.Bool("csv-header", false, "Print CSV headers")
	f.Bool("tsv-header", false, "Print TSV headers")
	f.StringVarP(
		&cliOpts.Format,
		"format",
		"f",
		"human-readable",
		`Output format [human-readable, simple, csv, tsv,
    json, jsonl, json-pretty], non-human readable formats
	show speeds in Mbps`,
	)
	f.BoolVar(
		&cliOpts.NoDownload,
		"no-download",
		false,
		"Do not perform download test",
	)
	f.BoolVar(
		&cliOpts.NoUpload,
		"no-upload",
		false,
		"Do not perform upload test",
	)
	f.BoolVar(&cliOpts.NoICMP, "no-icmp", false, "Do not use ICMP ping")
	f.BoolVar(
		&cliOpts.ForceHTTPS,
		"secure",
		false,
		`Use HTTPS instead of HTTP when communicating with
	LibreSpeed.org operated servers`,
	)
	f.BoolVar(
		&cliOpts.NoPreAllocate,
		"no-pre-allocate",
		false,
		`Do not pre allocate upload data. Pre allocation is
	enabled by default to improve upload performance. To
	support systems with insufficient memory, use this
	option to avoid out of memory errors.`,
	)
	f.IntVarP(
		&cliOpts.Concurrent,
		"concurrent",
		"c",
		3,
		"Concurrent HTTP requests being made",
	)
	f.IntVarP(
		&cliOpts.Chunks,
		"chunks",
		"C",
		100,
		`Chunks to download from server,
	chunk size depends on server configuration`,
	)
	f.BoolVarP(
		&cliOpts.Bytes,
		"bytes",
		"B",
		false,
		`Display values in bytes instead of bits. 
	Only applies to human readable output.`,
	)
	f.BoolVarP(
		&cliOpts.BinaryBase,
		"binary-base",
		"b",
		false,
		`Use a binary prefix (Kibibits, Mebibits, etc.) instead of decimal.
	Only applies to human readable output.`,
	)
	f.StringVarP(
		&cliOpts.DistanceUnit,
		"distance",
		"d",
		"km",
		`Change distance unit shown in ISP info, use 'mi' for miles,
	'km' for kilometres, 'NM' for nautical miles`,
	)
	f.IntVarP(
		&cliOpts.Duration,
		"duration",
		"D",
		15,
		"Upload and download test duration in seconds",
	)
	f.IntVarP(
		&cliOpts.UploadSize,
		"upload-size",
		"u",
		1024,
		"Size of payload being uploaded in KiB",
	)
	f.CountVarP(
		&cliOpts.LogVerbosity,
		"verbose",
		"v",
		"Logging verbosity. Specify multiple times for higher verbosity",
	)
	f.BoolVar(
		&cliOpts.Share,
		"share",
		false,
		`Generate and provide a URL to the LibreSpeed.org share results
image, not displayed with csv and tsv formats.`,
	)

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		if err := cliOpts.Complete(args); err != nil {
			return err
		}
		return cliOpts.Run(cmd, cmd.OutOrStderr())
	}

	return cmd
}
