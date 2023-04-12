package defs

import (
	"time"
)

// Report represents the output data fields in a nestable file data such as JSON.
type Report struct {
	Timestamp     time.Time `json:"timestamp"`
	Server        Server    `json:"server"`
	Client        Client    `json:"client"`
	BytesSent     int       `json:"bytes_sent"`
	BytesReceived int       `json:"bytes_received"`
	Ping          float64   `json:"ping"`
	Jitter        float64   `json:"jitter"`
	Upload        float64   `json:"upload"`
	Download      float64   `json:"download"`
	ShareLink     string    `json:"share_link"`
}

// FlatReport represents the output data fields in a flat file data such as CSV.
type FlatReport struct {
	Timestamp time.Time `csv:"Timestamp"`
	Name      string    `csv:"Server Name"`
	Address   string    `csv:"Address"`
	Ping      float64   `csv:"Ping"`
	Jitter    float64   `csv:"Jitter"`
	Download  float64   `csv:"Download"`
	Upload    float64   `csv:"Upload"`
	Share     string    `csv:"Share"`
	IP        string    `csv:"IP"`
}

// Client represents the speed test client's information
type Client struct {
	IPInfoResponse
}

func (r Report) GetFlatReport() FlatReport {
	var rep FlatReport

	rep.Timestamp = r.Timestamp
	rep.Name = r.Server.Name
	rep.Address = r.Server.Server
	rep.Ping = r.Ping
	rep.Jitter = r.Jitter
	rep.Download = r.Download
	rep.Upload = r.Upload
	rep.Share = r.ShareLink
	rep.IP = r.Client.IP

	return rep
}
