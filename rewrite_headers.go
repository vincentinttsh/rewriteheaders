//nolint
package rewriteheaders

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
)

// Rewrite holds one rewrite body configuration.
type Rewrite struct {
	Header      string `json:"header,omitempty"`
	Regex       string `json:"regex,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

// Config holds the plugin configuration.
type Config struct {
	Rewrites []Rewrite `json:"rewrites,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// LoggerINFO Main logger
var LoggerINFO = log.New(ioutil.Discard, "INFO: Fail2Ban: ", log.Ldate|log.Ltime|log.Lshortfile)

type rewrite struct {
	header      string
	regex       *regexp.Regexp
	replacement string
}

type rewriteBody struct {
	name     string
	next     http.Handler
	rewrites []rewrite
}

// New creates and returns a new rewrite body plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rewrites := make([]rewrite, len(config.Rewrites))
	LoggerINFO.SetOutput(os.Stdout)
	for i, rewriteConfig := range config.Rewrites {
		regex, err := regexp.Compile(rewriteConfig.Regex)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex %q: %w", rewriteConfig.Regex, err)
		}
		LoggerINFO.Printf("header %s: %q -> %s ", rewriteConfig.Header, regex, rewriteConfig.Replacement)
		rewrites[i] = rewrite{
			header:      rewriteConfig.Header,
			regex:       regex,
			replacement: rewriteConfig.Replacement,
		}
	}

	return &rewriteBody{
		name:     name,
		next:     next,
		rewrites: rewrites,
	}, nil
}

func (r *rewriteBody) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		ResponseWriter: rw,
		rewrites:       r.rewrites,
	}

	r.next.ServeHTTP(wrappedWriter, req)
}

type responseWriter struct {
	http.ResponseWriter
	rewrites []rewrite
}

func (r *responseWriter) WriteHeader(statusCode int) {
	for _, rewrite := range r.rewrites {
		headers := r.Header().Values(rewrite.header)

		if len(headers) == 0 {
			continue
		}

		r.Header().Del(rewrite.header)

		for _, header := range headers {
			value := rewrite.regex.ReplaceAllString(header, rewrite.replacement)
			r.Header().Add(rewrite.header, value)
		}
	}

	r.ResponseWriter.WriteHeader(statusCode)
}
