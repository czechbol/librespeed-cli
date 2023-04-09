package defs

type DistanceUnit string

const (
	Kilometres    DistanceUnit = "km"
	Miles         DistanceUnit = "mi"
	NauticalMiles DistanceUnit = "NM"
)

type TestOptions struct {
	Chunks          int
	Concurrent      int
	BinaryBase      bool
	Bytes           bool
	DistanceUnit    DistanceUnit
	Duration        int
	Network         string
	NoDownload      bool
	NoICMP          bool
	NoPreAllocate   bool
	NoUpload        bool
	Secure          bool
	ServerList      []Server
	Share           bool
	SkipCertVerify  bool
	SourceIP        string
	TelemetryServer TelemetryServer
	TelemetryExtra  string
	Timeout         int
	UploadSize      int
}

// Returns true if IPType is set to IPv4
func (testOpts TestOptions) IPv4() bool {
	return testOpts.Network == "ip4"
}

// Returns true if IPType is set to IPv6
func (testOpts TestOptions) IPv6() bool {
	return testOpts.Network == "ip6"
}

// Returns true if DistanceUnit is set to km
func (testOpts TestOptions) Kilometres() bool {
	return testOpts.DistanceUnit == DistanceUnit("km")
}

// Returns true if DistanceUnit is set to mi
func (testOpts TestOptions) Miles() bool {
	return testOpts.DistanceUnit == DistanceUnit("mi")
}

// Returns true if DistanceUnit is set to nm
func (testOpts TestOptions) NauticalMiles() bool {
	return testOpts.DistanceUnit == DistanceUnit("nm")
}
