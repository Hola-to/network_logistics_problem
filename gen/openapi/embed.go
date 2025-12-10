// Package openapi provides embedded OpenAPI specification.
package openapi

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
)

//go:embed api.swagger.json
var specBytes []byte

var (
	parsedSpec map[string]interface{}
	parseOnce  sync.Once
	parseErr   error
)

// ErrEmptySpec indicates that the embedded specification is empty.
var ErrEmptySpec = errors.New("openapi: embedded specification is empty")

// GetSpec returns the raw OpenAPI specification as bytes.
func GetSpec() ([]byte, error) {
	if len(specBytes) == 0 {
		return nil, ErrEmptySpec
	}
	return specBytes, nil
}

// MustGetSpec returns the specification or panics on error.
func MustGetSpec() []byte {
	spec, err := GetSpec()
	if err != nil {
		panic(err)
	}
	return spec
}

// GetSpecString returns the OpenAPI specification as a string.
func GetSpecString() (string, error) {
	spec, err := GetSpec()
	if err != nil {
		return "", err
	}
	return string(spec), nil
}

// GetSpecJSON returns the OpenAPI specification as parsed JSON map.
// Result is cached after first call.
func GetSpecJSON() (map[string]interface{}, error) {
	if len(specBytes) == 0 {
		return nil, ErrEmptySpec
	}

	parseOnce.Do(func() {
		parseErr = json.Unmarshal(specBytes, &parsedSpec)
	})

	if parseErr != nil {
		return nil, parseErr
	}

	return parsedSpec, nil
}

// Handler returns an http.Handler that serves the OpenAPI spec.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec, err := GetSpec()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(spec)
	})
}
