// Package v1alpha1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.3.0 DO NOT EDIT.
package v1alpha1

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"FTyN9qLd6OrT1f8FAAD//9z/+srcHAEA",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
