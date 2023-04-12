/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/czechbol/librespeedtest/defs"
	"github.com/sirupsen/logrus"
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
	listServers  bool
	csvHeader    bool
	networkCheck = map[string]bool{
		"ipv4": true,
		"ipv6": true,
	}
	formatCheck = map[string]bool{
		"human-readable": true,
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
	defs.TestOptions
	IncludeIDs    []int  `json:"include_ids,omitempty"`
	ExcludeIDs    []int  `json:"exclude_ids,omitempty"`
	Format        string `json:"format"`
	LocalServers  string `json:"local_servers,omitempty"`
	RemoteServers string `json:"remote_servers,omitempty"`
	LogVerbosity  int    `json:"-"`
}

func (testOpts *CLIOptions) Complete(log *logrus.Logger, args []string) error {
	if !networkCheck[testOpts.Network] {
		keys := make([]string, 0, len(networkCheck))
		for k := range networkCheck {
			keys = append(keys, k)
		}
		printKeys := "['" + strings.Join(keys, `','`) + `']`
		log.WithFields(logrus.Fields{
			"got":     testOpts.Network,
			"allowed": printKeys,
		}).Fatal("Invalid Argument")
	}
	if !formatCheck[testOpts.Format] {
		keys := make([]string, 0, len(formatCheck))
		for k := range formatCheck {
			keys = append(keys, k)
		}
		printKeys := "['" + strings.Join(keys, `','`) + `']`
		log.WithFields(logrus.Fields{
			"got":     testOpts.Format,
			"allowed": printKeys,
		}).Fatal("Invalid Argument")
	}
	if !distanceCheck[testOpts.DistanceUnit] {
		keys := make([]string, 0, len(distanceCheck))
		for k := range distanceCheck {
			keys = append(keys, k)
		}
		printKeys := "['" + strings.Join(keys, `','`) + `']`
		log.WithFields(logrus.Fields{
			"got":     testOpts.DistanceUnit,
			"allowed": printKeys,
		}).Fatal("Invalid Argument")
	}
	return nil
}

func (testOpts *CLIOptions) Run(cmd *cobra.Command, log *logrus.Logger, out io.Writer) error {
	log.SetLevel(logrus.Level(3 + testOpts.LogVerbosity))

	// Print CSV header and exit
	header, err := cmd.Flags().GetBool("csv-header")
	if err != nil {
		return fmt.Errorf("%w: Error getting boolean variable from cmd flags.", err)
	} else if header {
		log.Error("Print CSV Header here")
		return nil
	}

	var defaultServerList *[]defs.Server
	if defaultServerList, err = FetchServerList(serverListUrl); err != nil {
		log.Error("Unable to fetch server list")
	} else {
		testOpts.ServerList = *defaultServerList
	}

	var remoteServerList *[]defs.Server
	if testOpts.RemoteServers != "" {
		if remoteServerList, err = FetchServerList(testOpts.RemoteServers); err != nil {
			log.Error("Unable to fetch server list")
		} else {
			testOpts.ServerList = append(testOpts.ServerList, (*remoteServerList)...)
		}

	}

	var localServerList *[]defs.Server
	if testOpts.LocalServers != "" {
		if localServerList, err = GetLocalServerList(testOpts.LocalServers); err != nil {
			log.Error("Unable to local server list")
		} else {
			testOpts.ServerList = append(testOpts.ServerList, (*localServerList)...)
		}
	}

	// Print Server List and exit
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return fmt.Errorf("%w: Error getting boolean variable from cmd flags.", err)
	} else if list {
		log.Error("Print Server List here")
		return nil
	}
	res2B, _ := json.Marshal(testOpts)
	fmt.Println(string(res2B))

	return nil

}

func (testOpts *CLIOptions) CobraCommand(log *logrus.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     cmdUse,
		Short:   cmdShort,
		Long:    cmdLong,
		Version: "v0.0.1",
	}
	f := cmd.Flags()

	f.BoolP("list", "l", false, "Display a list of LibreSpeed.org servers")
	f.Bool("csv-header", false, "Print CSV headers")
	f.IntSliceVarP(&testOpts.IncludeIDs, "servers", "s", []int{}, "Specify comma separated SERVER IDs to test against.")
	f.IntSliceVarP(&testOpts.ExcludeIDs, "exclude", "e", []int{}, "Specify comma separated SERVER IDs to exclude from test.")
	f.StringVarP(&testOpts.Format, "format", "f", "human-readable", `Output format [human-readable, csv, tsv,
    json, jsonl, json-pretty], non-human readable formats
	show speeds in bits per second`)
	f.StringVarP(&testOpts.Network, "ip-version", "i", "ipv4", "IP version [ipv4, ipv6]")
	f.BoolVar(&testOpts.NoDownload, "no-download", false, "Do not perform download test")
	f.BoolVar(&testOpts.NoDownload, "no-upload", false, "Do not perform upload test")
	f.BoolVar(&testOpts.NoICMP, "no-icmp", false, "Do not use ICMP ping")
	f.BoolVar(&testOpts.NoPreAllocate, "no-pre-allocate", false, `Do not pre allocate upload data. Pre allocation is
	enabled by default to improve upload performance. To
	support systems with insufficient memory, use this
	option to avoid out of memory errors.`)
	f.IntVarP(&testOpts.Concurrent, "concurrent", "c", 3, "Concurrent HTTP requests being made")
	f.IntVarP(&testOpts.Chunks, "chunks", "C", 100, `Chunks to download from server,
	chunk size depends on server configuration`)
	f.BoolVarP(&testOpts.Bytes, "bytes", "B", false, `Display values in bytes instead of bits. 
	Only applies to human readable output.`)
	f.BoolVarP(&testOpts.BinaryBase, "binary-base", "b", false, `Use a binary prefix (Kibibits, Mebibits, etc.) instead of decimal.
	Only applies to human readable output.`)
	f.StringVarP(&testOpts.DistanceUnit, "distance", "d", "km", `Change distance unit shown in ISP info, use 'mi' for miles,
	'km' for kilometres, 'NM' for nautical miles`)
	f.StringVarP(&testOpts.LocalServers, "local-servers", "L", "", `Use an alternative server list from local JSON file`)
	f.StringVarP(&testOpts.RemoteServers, "remote-servers", "R", "", `Use an alternative server list from a remote JSON file by URL`)
	f.IntVarP(&testOpts.Timeout, "timeout", "t", 15, "HTTP TIMEOUT in seconds")
	f.IntVarP(&testOpts.Duration, "duration", "D", 15, "Upload and download test duration in seconds")
	f.IntVarP(&testOpts.UploadSize, "upload-size", "u", 1024, "Size of payload being uploaded in KiB")
	f.CountVarP(&testOpts.LogVerbosity, "verbose", "v", "Logging verbosity. Specify multiple times for higher verbosity")

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		if err := testOpts.Complete(log, args); err != nil {
			return err
		}
		return testOpts.Run(cmd, log, cmd.OutOrStdout())
	}

	return cmd
}
