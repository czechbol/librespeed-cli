package defs

type TestOptions struct {
	Chunks          int             `json:"chunks"`
	Concurrent      int             `json:"concurrent"`
	BinaryBase      bool            `json:"binary_base,omitempty"`
	Bytes           bool            `json:"bytes,omitempty"`
	DistanceUnit    string          `json:"distance_unit"`
	Duration        int             `json:"duration"`
	Network         string          `json:"network"`
	NoDownload      bool            `json:"no_download,omitempty"`
	NoICMP          bool            `json:"no_icmp,omitempty"`
	NoPreAllocate   bool            `json:"no_pre_allocate,omitempty"`
	NoUpload        bool            `json:"no_upload,omitempty"`
	Secure          bool            `json:"secure,omitempty"`
	ServerList      []Server        `json:"server_list,omitempty"`
	Share           bool            `json:"share,omitempty"`
	SkipCertVerify  bool            `json:"skip_cert_verify,omitempty"`
	SourceIP        string          `json:"source_ip,omitempty"`
	TelemetryServer TelemetryServer `json:"telemetry_server"`
	TelemetryExtra  string          `json:"telemetry_extra,omitempty"`
	Timeout         int             `json:"timeout"`
	UploadSize      int             `json:"upload_size"`
}
