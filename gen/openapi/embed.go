package openapi

import (
	"embed"
)

//go:embed api.swagger.json
var content embed.FS

// GetSpec возвращает содержимое OpenAPI спецификации
func GetSpec() ([]byte, error) {
	return content.ReadFile("api.swagger.json")
}

// MustGetSpec возвращает спецификацию или паникует
func MustGetSpec() []byte {
	data, err := GetSpec()
	if err != nil {
		panic("failed to load OpenAPI spec: " + err.Error())
	}
	return data
}
