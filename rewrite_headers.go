package rewriteheaders

import (
	"context"
	"fmt"
	"net/http"
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

type rewrite struct {
	header      string
	regex       *regexp.Regexp
	replacement []byte
}

type rewriteHeader struct {
	name     string
	next     http.Handler
	rewrites []rewrite
}

// New creates and returns a new rewrite body plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rewrites := make([]rewrite, len(config.Rewrites))
	for i, rewriteConfig := range config.Rewrites {
		regex, err := regexp.Compile(rewriteConfig.Regex)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex %q: %w", rewriteConfig.Regex, err)
		}
		rewrites[i] = rewrite{
			header:      rewriteConfig.Header,
			regex:       regex,
			replacement: []byte(rewriteConfig.Replacement),
		}
	}

	return &rewriteHeader{
		name:     name,
		next:     next,
		rewrites: rewrites,
	}, nil
}

func (r *rewriteHeader) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	r.next.ServeHTTP(rw, req)

	for _, rewrite := range r.rewrites {
		headers := rw.Header().Get(rewrite.header)

		if len(headers) == 0 {
			continue
		}

		rw.Header().Del(rewrite.header)

		value := rewrite.regex.ReplaceAll([]byte(headers), rewrite.replacement)
		rw.Header().Add(rewrite.header, string(value))
	}
}
