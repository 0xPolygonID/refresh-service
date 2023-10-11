package flexiblehttp

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type settings struct {
	TimeExpiration time.Duration `yaml:"timeExpiration"`
}

type provider struct {
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
}

type requestSchema struct {
	Params  map[string]string `yaml:"params"`
	Headers map[string]string `yaml:"headers"`
}

type responseSchema struct {
	Type       string                  `yaml:"type"`
	Properties map[string]matchedField `yaml:"properties"`
}

type matchedField struct {
	Type    string `yaml:"type"`
	MatchTo string `yaml:"match"`
}

type FlexibleHTTP struct {
	httpcli        *http.Client
	Settings       settings       `yaml:"settings"`
	Provider       provider       `yaml:"provider"`
	RequestSchema  requestSchema  `yaml:"requestSchema"`
	ResponseSchema responseSchema `yaml:"responseSchema"`
}

func (fh *FlexibleHTTP) Provide(credentialSubject map[string]interface{}) (map[string]interface{}, error) {
	req, err := fh.BuildRequest(credentialSubject)
	if err != nil {
		return nil, err
	}
	resp, err := fh.httpcli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	return fh.DecodeResponse(resp.Body, fh.ResponseSchema.Type)
}

func (fh FlexibleHTTP) BuildRequest(credentialSubject map[string]interface{}) (*http.Request, error) {
	u, err := url.Parse(fh.Provider.URL)
	if err != nil {
		return nil, err
	}

	urlParts := strings.Split(u.Path, "/")
	for i, part := range urlParts {
		if isPlaceholder(part) {
			value, err := findPlaceholderValue(part, credentialSubject)
			if err != nil {
				return nil, err
			}
			urlParts[i] = fmt.Sprintf("%v", value)
		}
	}
	u.Path = strings.Join(urlParts, "/")

	q := u.Query()
	for argK, argV := range fh.RequestSchema.Params {
		if isPlaceholder(argV) {
			value, err := findPlaceholderValue(argV, credentialSubject)
			if err != nil {
				return nil, err
			}
			argV = fmt.Sprintf("%v", value)
		}
		q.Add(argK, argV)
	}
	u.RawQuery = q.Encode()

	request, err := http.NewRequest(
		fh.Provider.Method,
		u.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	for headerK, headerV := range fh.RequestSchema.Headers {
		request.Header.Add(headerK, headerV)
	}

	return request, nil
}

func (fh FlexibleHTTP) DecodeResponse(body io.Reader, responseType string) (map[string]interface{}, error) {
	if fh.ResponseSchema.Type != responseType {
		return nil, fmt.Errorf("response type is not supported %s", responseType)
	}
	var response map[string]interface{}
	if err := yaml.NewDecoder(body).Decode(&response); err != nil {
		return nil, err
	}

	parsedFields := make(map[string]interface{})
	for propertyKey, propertyValue := range fh.ResponseSchema.Properties {
		parts := strings.Split(propertyKey, ".")
		for i, part := range parts {
			tragetKey, targetIndex := processKey(part)
			if tragetKey == "" {
				return nil, fmt.Errorf("invalid key %s", part)
			}

			v, ok := response[tragetKey]
			if !ok {
				return nil, fmt.Errorf("not found field %s in response", parts[:i+1])
			}
			switch v := v.(type) {
			case map[string]interface{}:
				response = v
			case []interface{}:
				if targetIndex == -1 {
					return nil, fmt.Errorf("not found index for %s", part)
				}
				if targetIndex >= len(v) {
					return nil, fmt.Errorf("index out of range for %s", part)
				}
				tmp := v[targetIndex]
				switch tmp := tmp.(type) {
				case map[string]interface{}:
					response = tmp
				default:
					p := strings.Split(propertyValue.MatchTo, ".")
					if len(p) != 2 {
						return nil, fmt.Errorf("invalid match field for %s", propertyKey)
					}
					parsedFields[p[1]] = v[targetIndex]
				}
			default:
				p := strings.Split(propertyValue.MatchTo, ".")
				if len(p) != 2 {
					return nil, fmt.Errorf("invalid match field for %s", propertyKey)
				}
				parsedFields[p[1]] = v
			}
		}
	}

	return parsedFields, nil
}

func isPlaceholder(v string) bool {
	return strings.HasPrefix(v, "{{") && strings.HasSuffix(v, "}}")
}

func processKey(key string) (string, int) {
	startIdx := strings.Index(key, "[")
	endIdx := strings.Index(key, "]")
	if startIdx == -1 || endIdx == -1 {
		return key, -1
	}
	arrayIdx, err := strconv.Atoi(key[startIdx+1 : endIdx])
	if err != nil {
		return key, -1
	}
	return key[:startIdx], arrayIdx
}

func findPlaceholderValue(placeholder string, values map[string]interface{}) (interface{}, error) {
	placeHolder := strings.Trim(placeholder, "{ }")
	pair := strings.Split(placeHolder, ".")
	if len(pair) != 2 {
		return nil, fmt.Errorf("invalid placeholder fromat: %s", placeHolder)
	}
	v, ok := values[pair[1]]
	if !ok {
		return nil, fmt.Errorf("not found value for placeholder: %s", placeHolder)
	}
	return v, nil
}

func custToType(v interface{}, toType string) interface{} {
	switch toType {
	case "string":
		v = fmt.Sprintf("%v", v)
	case "int":
		v = int(v.(float64))
	case "float":
		v = float32(v.(float64))
	case "bool":
		v = v.(bool)
	}
	return v
}
