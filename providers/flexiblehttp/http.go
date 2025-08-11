package flexiblehttp

import (
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	ErrInvalidRequestSchema  = errors.New("invalid request schema")
	ErrInvalidResponseSchema = errors.New("invalid response schema")
	ErrDataProviderIssue     = errors.New("data provider issue")
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
		return nil, errors.Wrap(ErrInvalidRequestSchema, err.Error())
	}

	resp, err := fh.httpcli.Do(req)
	if err != nil {
		return nil, errors.Wrapf(ErrDataProviderIssue,
			"failed http request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.Wrapf(ErrDataProviderIssue,
			"unexpected status code '%d'", resp.StatusCode)
	}
	response := map[string]interface{}{}
	if err := yaml.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrapf(ErrDataProviderIssue, "failed to decode response: %v", err)
	}

	decodedResponse, err := fh.DecodeResponse(response)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidResponseSchema,
			"failed to decode response by response schema: %v", err)
	}
	return decodedResponse, nil
}

func (fh *FlexibleHTTP) BuildRequest(credentialSubject map[string]interface{}) (*http.Request, error) {
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
		http.NoBody,
	)
	if err != nil {
		return nil, err
	}
	for headerK, headerV := range fh.RequestSchema.Headers {
		request.Header.Add(headerK, headerV)
	}

	return request, nil
}

func (fh *FlexibleHTTP) DecodeResponse(response map[string]interface{}) (map[string]interface{}, error) {
	parsedFields := make(map[string]interface{})
	for propertyKey, propertyValue := range fh.ResponseSchema.Properties {
		parts := strings.Split(propertyKey, ".")
		for i, part := range parts {
			tragetKey, targetIndex := processKey(part)
			if tragetKey == "" {
				return nil, errors.Errorf("invalid key '%s'", part)
			}

			v, ok := response[tragetKey]
			if !ok {
				return nil, errors.Errorf("not found field '%s' in response", parts[:i+1])
			}
			switch v := v.(type) {
			case map[string]interface{}:
				response = v
			case []interface{}:
				if targetIndex == -1 {
					return nil, errors.Errorf("not found index for '%s'", part)
				}
				if targetIndex >= len(v) {
					return nil, errors.Errorf("index out of range for '%s'", part)
				}
				tmp := v[targetIndex]
				switch tmp := tmp.(type) {
				case map[string]interface{}:
					response = tmp
				default:
					p := strings.Split(propertyValue.MatchTo, ".")
					if len(p) != 2 {
						return nil, errors.Errorf("invalid match field for '%s'", propertyKey)
					}
					var err error
					parsedFields[p[1]], err = castToType(v[targetIndex], propertyValue.Type)
					if err != nil {
						return nil, err
					}
				}
			default:
				p := strings.Split(propertyValue.MatchTo, ".")
				if len(p) != 2 {
					return nil, errors.Errorf("invalid match field for '%s'", propertyKey)
				}
				var err error
				parsedFields[p[1]], err = castToType(v, propertyValue.Type)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return parsedFields, nil
}

func isPlaceholder(v string) bool {
	return strings.HasPrefix(v, "{{") && strings.HasSuffix(v, "}}")
}

// nolint:gocritic // clear with named return
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
		return nil, errors.Errorf("invalid placeholder fromat: '%s'", placeHolder)
	}
	v, ok := values[pair[1]]
	if !ok {
		return nil, errors.Errorf("not found value for placeholder: '%s'", placeHolder)
	}
	return v, nil
}

func castToType(v interface{}, toType string) (interface{}, error) {
	switch valueType := v.(type) {
	case string:
		return stringToType(valueType, toType)
	case float64:
		return float64ToType(valueType, toType)
	case bool:
		return booleanToType(valueType, toType)
	default:
		return nil, errors.Errorf("invalid type '%T' from JSON response", v)
	}
}

func stringToType(value, convertType string) (interface{}, error) {
	switch convertType {
	case "string":
		return value, nil
	case "integer":
		return strconv.Atoi(value)
	case "double", "number":
		return strconv.ParseFloat(value, 64)
	case "boolean", "bool":
		return strconv.ParseBool(value)
	default:
		return nil, errors.Errorf("not possible convert string to '%s'", convertType)
	}
}

func float64ToType(value float64, convertType string) (interface{}, error) {
	switch convertType {
	case "string":
		f := new(big.Float).SetFloat64(value)
		return f.String(), nil
	case "integer":
		return doubleToInt(value)
	case "float":
		return value, nil
	default:
		return nil, errors.Errorf("not possible convert float64 to '%s'", convertType)
	}
}

func doubleToInt(v float64) (int, error) {
	r := new(big.Rat).SetFloat64(v)
	if r.Denom().Cmp(big.NewInt(1)) != 0 {
		return 0, errors.New("value is too big to be converted to float64")
	}
	if r.Num().Cmp(big.NewInt(int64(v))) != 0 {
		return 0, errors.New("value is too big to be converted to float64")
	}
	return int(r.Num().Int64()), nil
}

func booleanToType(value bool, convertType string) (interface{}, error) {
	switch convertType {
	case "boolean", "bool":
		return value, nil
	case "string":
		return strconv.FormatBool(value), nil
	case "integer":
		if value {
			return 1, nil
		}
		return 0, nil
	default:
		return nil, errors.Errorf("not possible convert bool to '%s'", convertType)
	}
}
