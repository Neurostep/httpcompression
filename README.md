# Golang HTTP server middleware for gzip/brotli/zstandard compression

[![Build status](https://github.com/CAFxX/httpcompression/workflows/Build/badge.svg)](https://github.com/CAFxX/httpcompression/actions)
[![codecov](https://codecov.io/gh/CAFxX/httpcompression/branch/master/graph/badge.svg)](https://codecov.io/gh/CAFxX/httpcompression)
[![Go Report](https://goreportcard.com/badge/github.com/CAFxX/httpcompression)](https://goreportcard.com/report/github.com/CAFxX/httpcompression)
[![CodeFactor](https://www.codefactor.io/repository/github/cafxx/httpcompression/badge)](https://www.codefactor.io/repository/github/cafxx/httpcompression)
[![Go Reference](https://pkg.go.dev/badge/github.com/CAFxX/httpcompression.svg)](https://pkg.go.dev/github.com/CAFxX/httpcompression)

This is a small Go package which wraps HTTP handlers to transparently compress
response bodies using zstd, brotli or gzip - for clients which support them. Although
it's usually simpler to leave that to a reverse proxy (like nginx or Varnish),
this package is useful when that is undesirable. In addition, this package allows
users to extend it by plugging in third-party or custom compression encoders.

**Note: This package was recently forked from NYTimes/gziphandler.
Maintaining drop-in compatibility is not a goal of this fork, as the scope of this fork
is significantly wider than the original package.**

:warning: As we have not reached 1.0 yet, API is still subject to changes.

## Features

- gzip, brotli, and zstd compression by default, alternate (faster) gzip, zstd implementations are optional
- Apply compression only if response body size is greater than a threshold
- Apply compression only to a allowlist/denylist of MIME content types
- Define encoding priority (e.g. give brotli a higher priority than gzip)
- Control whether the client or the server defines the encoder priority
- Plug in third-party/custom compression schemes or implementations
- Custom dictionary compression for zstd
- Low memory alliocations via transparent encoder reuse

## Install

```bash
go get github.com/CAFxX/httpcompression
```

## Usage

Call `httpcompression.DefaultAdapter` to get an adapter that can be used to wrap
any handler (an object which implements the `http.Handler` interface),
to transparently provide response body compression.
Note that `httpcompression` automatically compresses using Zstandard, Brotli, and Gzip
depending on the capabilities of the client (`Accept-Encoding`)
and the configuration of this handler (by default, Zstandard, Brotli and gzip are
all enabled and, conditional on client support, used in that order of preference).

As a simple example:

```go
package main

import (
    "io"
    "net/http"
    "github.com/CAFxX/httpcompression"
)

func main() {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain")
        io.WriteString(w, "Hello, World")
    })
    compress, _ := httpcompression.DefaultAdapter() // Use the default configuration
    http.Handle("/", compress(handler))
    http.ListenAndServe("0.0.0.0:8080", nil)
}
```

### Pluggable compressors

It is possible to use custom compressor implementations by specifying a `CompressorProvider`
for each of the encodings the adapter should support. This also allows to support arbitrary
`Content-Encoding` schemes (e.g. `lzma`, or zstd with a static dictionary - see the
[examples](example_test.go)).

```go
pgz, err := httpcompression.Compressor("gzip", 0, pgzip.New(pgzip.Options{Level: 6}))
if err != nil {
    log.Fatal(err)
}
compress, err := httpcompression.Adapter(
    // use klauspost/pgzip as compressor for the "gzip" content-encoding, with priority 0
    pgz,
)
if err != nil {
    log.Fatal(err)
}
http.Handle("/", compress(handler))
```

The `contrib/` directory contains a number of bundled implementations that are ready for use:

| `Content-Encoding` | Provider package                                                                                             | Implementation package                                                      | Notes                                                  |
|--------------------|--------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------|--------------------------------------------------------|
| `gzip`             | [contrib/klauspost/gzip](https://pkg.go.dev/github.com/CAFxX/httpcompression/contrib/klauspost/gzip)         | [github.com/klauspost/compress/gzip](https://github.com/klauspost/compress) | Go implementation, faster than compress/gzip           |
| `gzip`             | [contrib/klauspost/pgzip](https://pkg.go.dev/github.com/CAFxX/httpcompression/contrib/klauspost/pgzip)       | [github.com/klauspost/pgzip](https://github.com/klauspost/pgzip)            | Go implementation, parallel compression                |
| `zstd`             | [contrib/klauspost/zstd](https://pkg.go.dev/github.com/CAFxX/httpcompression/contrib/klauspost/zstd)         | [github.com/klauspost/compress/zstd](https://github.com/klauspost/compress) | Go implementation                                      |
| `zstd`             | [contrib/valyala/gozstd](https://pkg.go.dev/github.com/CAFxX/httpcompression/contrib/valyala/gozstd)         | [github.com/valyala/gozstd](https://github.com/valyala/gozstd)              | cgo wrapper                                            |
| `brotli`           | [contrib/andybalholm/brotli](https://pkg.go.dev/github.com/CAFxX/httpcompression/contrib/andybalholm/brotli) | [github.com/andybalholm/brotli](https://github.com/andybalholm/brotli)      | Go implementation                                      |
| `brotli`           | [contrib/google/cbrotli](https://pkg.go.dev/github.com/CAFxX/httpcompression/contrib/google/cbrotli)         | [github.com/google/brotli](https://github.com/google/brotli)                | cgo wrapper, requires brotli libraries to be installed |

## Benchmark

See the [benchmark results](results.md) to get an idea of the relative performance and
compression efficiency of gzip, brotli and zstd in the current implementation.

## TODO

- Add dictionary support to gzip and brotli (zstd already supports it)
- Allow to choose dictionary based on content-type
- Provide additional implementations based on the bindings to the original native implementations
- Add compressed payload caching (if the same payload has already been compressed and is present in the cache, skip compression)
- Add write buffering (compress larger chunks at once)
- Add decompression (if the payload is already compressed but the client supports better algorithms, or does not support a certain algorithm)

## License

[Apache 2.0][license].

[docs]:     https://godoc.org/github.com/CAFxX/httpcompression
[license]:  https://github.com/CAFxX/httpcompression/blob/master/LICENSE
