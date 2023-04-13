# LibreSpeedtest

![LibreSpeed Logo](https://github.com/librespeed/speedtest/blob/master/.logo/logo3.png?raw=true)

Don't have a GUI but want to use LibreSpeed servers to test your Internet speed? 🚀

`librespeedtest` comes to rescue!

This is a command line interface for LibreSpeed speed test backends, written in Go.

## Notice

This is a fork of [librespeed-cli](https://github.com/librespeed/speedtest-cli) with reduced functionality. It brings the ability to be imported into other Go projects so you can run speedtests inside your own project.

## Features

- Ping
- Jitter
- Download
- Upload
- IP address
- ISP Information
- Result sharing (telemetry) *[optional]*
- Tested with PHP and Go backends

[![asciicast](https://asciinema.org/a/R0LQsbZBKd6i0NGqdotOO7Icr.svg)](https://asciinema.org/a/R0LQsbZBKd6i0NGqdotOO7Icr)
## Requirements for compiling

- Go 1.20+

## Runtime requirements

- Any [Go supported platforms](https://github.com/golang/go/wiki/MinimumRequirements)

## Use prebuilt binaries

If you don't want to build `librespeedtest` yourself, you can find different binaries compiled for various platforms in
the [releases page](https://github.com/czechbol/librespeedtest/releases).

## Install with Go

If you have Go installed, installing the latest version is as simple as:

  ```shell script
  go install github.com/czechbol/librespeedtest@latest
  ```

## Building `librespeedtest`

1. First, you'll have to install Go (at least version 1.20). For Windows users, [you can download an installer from golang.org](https://golang.org/dl/).
For Linux users, you can use either the archive from golang.org, or install from your distribution's package manager.

    For example, Arch Linux:

    ```shell script
    pacman -S go
    ```

2. Then, clone the repository:

    ```shell script
    git clone -b v2.0.0 https://github.com/czechbol/librespeedtest
    ```

3. After you have Go installed on your system (and added to your `$PATH` if you're using the archive from golang.org), you
can now proceed to build `librespeedtest` with the build script:

    ```shell script
    cd speedtest-cli
    ./build.sh
    ```

    If you want to build for another operating system or system architecture, use the `GOOS` and `GOARCH` environment
    variables. Run `go tool dist list` to get a list of possible combinations of `GOOS` and `GOARCH`.

    Note: Technically, the CLI can be compiled with older Go versions that support Go modules, with `GO111MODULE=on`
    set. If you're compiling with an older Go runtime, you might have to change the Go version in `go.mod`.

    ```shell script
    # Let's say we're building for 64-bit Windows on Linux
    GOOS=windows GOARCH=amd64 ./build.sh
    ```

4. When the build script finishes, if everything went smoothly, you can find the built binary under directory `out`.

    ```shell script
    $ ls out
    librespeedtest-windows-amd64.exe
    ```

5. Now you can use the `librespeedtest` and test your Internet speed!


## Usage

You can see the full list of supported options with `librespeedtest -h`:

```shell script
$ librespeedtest -h
librespeedtest - Test your Internet speed with LibreSpeed 🚀

No Flash, No Java, No Websocket, No Bullshit. 
Librespeed is an open-source speedtest for measuring your Internet speed.

Usage:
  librespeedtest [flags]

Flags:
  -b, --binary-base       Use a binary prefix (Kibibits, Mebibits, etc.) instead of decimal.
                                Only applies to human readable output.
  -B, --bytes             Display values in bytes instead of bits. 
                                Only applies to human readable output.
  -C, --chunks int        Chunks to download from server,
                                chunk size depends on server configuration (default 100)
  -c, --concurrent int    Concurrent HTTP requests being made (default 3)
      --csv-header        Print CSV headers
  -d, --distance string   Change distance unit shown in ISP info, use 'mi' for miles,
                                'km' for kilometres, 'NM' for nautical miles (default "km")
  -D, --duration int      Upload and download test duration in seconds (default 15)
  -f, --format string     Output format [human-readable, simple, csv, tsv,
                              json, jsonl, json-pretty], non-human readable formats
                                show speeds in Mbps (default "human-readable")
  -h, --help              help for librespeedtest
  -l, --list              Display a list of LibreSpeed.org servers
      --no-download       Do not perform download test
      --no-icmp           Do not use ICMP ping
      --no-pre-allocate   Do not pre allocate upload data. Pre allocation is
                                enabled by default to improve upload performance. To
                                support systems with insufficient memory, use this
                                option to avoid out of memory errors.
      --no-upload         Do not perform upload test
      --secure            Use HTTPS instead of HTTP when communicating with
                                LibreSpeed.org operated servers
      --share             Generate and provide a URL to the LibreSpeed.org share results
                          image, not displayed with csv and tsv formats.
      --tsv-header        Print TSV headers
  -u, --upload-size int   Size of payload being uploaded in KiB (default 1024)
  -v, --verbose count     Logging verbosity. Specify multiple times for higher verbosity
      --version           version for librespeedtest
```

## Bugs?

Although we have tested the cli, it's still in its early days. Please open an issue if you encounter any bugs, or even
better, submit a PR.

## How to contribute

If you have some good ideas on improving `librespeedtest`, you can always submit a PR via GitHub.

## License

`librespeedtest` is licensed under [GNU Lesser General Public License v3.0](https://github.com/czechbol/librespeedtest/blob/master/LICENSE)
