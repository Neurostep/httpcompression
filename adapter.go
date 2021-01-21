package httpcompression // import "github.com/CAFxX/httpcompression"

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"sync"

	"github.com/CAFxX/httpcompression/contrib/andybalholm/brotli"
	_brotli "github.com/andybalholm/brotli"
)

const (
	vary            = "Vary"
	acceptEncoding  = "Accept-Encoding"
	contentEncoding = "Content-Encoding"
	contentType     = "Content-Type"
	contentLength   = "Content-Length"
	gzipEncoding    = "gzip"
	brotliEncoding  = "br"
)

type codings map[string]float64

const (
	// DefaultMinSize is the default minimum size for which we enable compression.
	// 20 is a very conservative default borrowed from nginx: you will probably want
	// to measure if a higher minimum size improves performance for your workloads.
	DefaultMinSize = 20
)

// Adapter returns a HTTP handler wrapping function (a.k.a. middleware)
// which can be used to wrap an HTTP handler to transparently compress the response
// body if the client supports it (via the Accept-Encoding header).
// It is possible to pass one or more options to modify the middleware configuration.
// An error will be returned if invalid options are given.
func Adapter(opts ...Option) (func(http.Handler) http.Handler, error) {
	c := config{
		prefer:     PreferServer,
		compressor: comps{},
	}
	for _, o := range opts {
		err := o(&c)
		if err != nil {
			return nil, err
		}
	}

	bufPool := &sync.Pool{}
	writerPool := &sync.Pool{}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			addVaryHeader(w.Header(), acceptEncoding)

			accept := parseEncodings(r.Header.Get(acceptEncoding))
			common := acceptedCompression(accept, c.compressor)
			if len(common) == 0 {
				h.ServeHTTP(w, r)
				return
			}

			gw, _ := writerPool.Get().(*compressWriter)
			if gw == nil {
				gw = &compressWriter{}
			}
			*gw = compressWriter{
				ResponseWriter: w,
				config:         c,
				accept:         accept,
				common:         common,
				pool:           bufPool,
			}
			defer func() {
				_ = gw.Close() // expose the error
				*gw = compressWriter{}
				writerPool.Put(gw)
			}()

			if _, ok := w.(http.CloseNotifier); ok {
				w = compressWriterWithCloseNotify{gw}
			} else {
				w = gw
			}

			h.ServeHTTP(w, r)
		})
	}, nil
}

func addVaryHeader(h http.Header, value string) {
	for _, v := range h.Values(vary) {
		if v == value {
			return
		}
	}
	h.Add(vary, value)
}

// DefaultAdapter is like Adapter, but it includes sane defaults for general usage.
// The provided opts override the defaults.
// The defaults are not guaranteed to remain constant over time: if you want to avoid this
// use Adapter directly.
func DefaultAdapter(opts ...Option) (func(http.Handler) http.Handler, error) {
	defaults := []Option{
		GzipCompressionLevel(gzip.DefaultCompression),
		BrotliCompressionLevel(_brotli.DefaultCompression),
		MinSize(DefaultMinSize),
	}
	opts = append(defaults, opts...)
	return Adapter(opts...)
}

// Used for functional configuration.
type config struct {
	minSize      int                 // Specifies the minimum response size to gzip. If the response length is bigger than this value, it is compressed.
	contentTypes []parsedContentType // Only compress if the response is one of these content-types. All are accepted if empty.
	blacklist    bool
	prefer       PreferType
	compressor   comps
}

type comps map[string]comp

type comp struct {
	comp     CompressorProvider
	priority int
}

// Option can be passed to Handler to control its configuration.
type Option func(c *config) error

// MinSize is an option that controls the minimum size of payloads that
// should be compressed. The default is DefaultMinSize.
func MinSize(size int) Option {
	return func(c *config) error {
		if size < 0 {
			return fmt.Errorf("minimum size can not be negative: %d", size)
		}
		c.minSize = size
		return nil
	}
}

// GzipCompressionLevel is an option that controls the Gzip compression
// level to be used when compressing payloads.
// The default is gzip.DefaultCompression.
func GzipCompressionLevel(level int) Option {
	c, err := NewDefaultGzipCompressor(level)
	if err != nil {
		return errorOption(err)
	}
	return GzipCompressor(c)
}

// BrotliCompressionLevel is an option that controls the Brotli compression
// level to be used when compressing payloads.
// The default is 3 (the same default used in the reference brotli C
// implementation).
func BrotliCompressionLevel(level int) Option {
	c, err := brotli.New(_brotli.WriterOptions{Quality: level})
	if err != nil {
		return errorOption(err)
	}
	return BrotliCompressor(c)
}

// GzipCompressor is an option to specify a custom compressor factory for Gzip.
func GzipCompressor(g CompressorProvider) Option {
	return Compressor(gzipEncoding, 0, g)
}

// BrotliCompressor is an option to specify a custom compressor factory for Brotli.
func BrotliCompressor(b CompressorProvider) Option {
	return Compressor(brotliEncoding, 1, b)
}

func errorOption(err error) Option {
	return func(_ *config) error {
		return err
	}
}
