package common

import (
	"fmt"
	"golang.org/x/exp/slices"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type RestURLField struct {
	Name        string
	Description string
	Default     string
}

type RestURLStats struct {
	sync.Mutex
	Count       int          `json:"count,omitempty"`
	SumDuration DurationJSON `json:"sumDuration,omitempty"`
	MinDuration DurationJSON `json:"minDuration,omitempty"`
	MaxDuration DurationJSON `json:"maxDuration,omitempty"`
}

type RestURL struct {
	Description string         `json:"description,omitempty"`
	Method      string         `json:"method,omitempty"`
	Resource    string         `json:"resoure,omitempty"`
	Consumes    []string       `json:"consumes,omitempty"`
	Produces    []string       `json:"produces,omitempty"`
	Success     []int          `json:"success,omitempty"`
	Failure     []int          `json:"failure,omitempty"`
	Headers     []RestURLField `json:"headers,omitempty"`
	Params      []RestURLField `json:"params,omitempty"`
	Stats       RestURLStats   `json:"stats,omitempty"`
}

func NewRestURL(method string, resource string) *RestURL {
	return &RestURL{Method: method, Resource: resource}
}

func (restUrl *RestURL) MuxString() string {
	return fmt.Sprintf("%s %s", restUrl.Method, restUrl.Resource)
}

func (restUrl *RestURL) Format(args ...any) string {
	s := restUrl.Resource

	if strings.Contains(s, "{") {
		regex := regexp.MustCompile("{.*?\\}")

		s = regex.ReplaceAllString(s, "%v")
		s = fmt.Sprintf(s, args...)
	}

	return s
}

func (restUrl *RestURL) URL(address string, args ...any) string {
	return fmt.Sprintf("%s%s", address, restUrl.Format(args...))
}

func (restUrl *RestURL) Validate(r *http.Request) error {
	if restUrl.Method != r.Method {
		return fmt.Errorf("invalid HTTP method")
	}

	rPaths := Split(r.URL.Path, "/")
	uPaths := Split(restUrl.Resource, "/")

	if !strings.Contains(restUrl.Resource, "*") {
		if len(rPaths) != len(uPaths) {
			return fmt.Errorf("invalid amount of HTTP request path")
		}
	}

	for _, header := range restUrl.Headers {
		if r.Header.Get(header.Name) == "" {
			if header.Default == "" {
				return fmt.Errorf("missing HTTP header: %s", header.Name)
			}
		}
	}

	for _, value := range restUrl.Params {
		if !r.URL.Query().Has(value.Name) {
			if value.Default == "" {
				return fmt.Errorf("missing HTTP value: %s", value.Name)
			}
		}
	}

	return nil
}

func (restUrl *RestURL) Header(r *http.Request, name string) string {
	v := r.Header.Get(name)
	if v == "" {
		p := slices.IndexFunc(restUrl.Headers, func(field RestURLField) bool {
			return field.Name == name
		})

		if p != -1 {
			v = restUrl.Headers[p].Default
		}
	}

	return v
}

func (restUrl *RestURL) Param(r *http.Request, name string) string {
	q := r.URL.Query()
	v := q.Get(name)
	if v == "" {
		p := slices.IndexFunc(restUrl.Params, func(field RestURLField) bool {
			return field.Name == name
		})

		if p != -1 {
			v = restUrl.Params[p].Default
		}
	}

	return v
}

func (restUrl *RestURL) UpdateStats(start time.Time) {
	d := time.Since(start)

	restUrl.Stats.Lock()
	defer func() {
		restUrl.Stats.Unlock()
	}()

	restUrl.Stats.Count++
	restUrl.Stats.SumDuration.Duration += d
	if restUrl.Stats.Count == 1 {
		restUrl.Stats.MinDuration.Duration = d
		restUrl.Stats.MaxDuration.Duration = d
	} else {
		restUrl.Stats.MinDuration.Duration = Min(restUrl.Stats.MinDuration.Duration, d)
		restUrl.Stats.MaxDuration.Duration = Max(restUrl.Stats.MaxDuration.Duration, d)
	}

	DebugFunc("restUrl.UpdateStats: %+v", restUrl.Stats)
}

func (restUrl *RestURL) SwaggerInfo() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("@Title %s\n", restUrl.Resource))
	sb.WriteString(fmt.Sprintf("@Description %s\n", restUrl.Description))
	sb.WriteString(fmt.Sprintf("@Accept %s\n", strings.Join(restUrl.Consumes, " ")))

	return sb.String()
}
