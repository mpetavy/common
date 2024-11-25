package common

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

type RestURLField struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
	Mandatory   bool   `json:"mandatory,omitempty"`
}

type RestURLStats struct {
	Count       int          `json:"count,omitempty"`
	SumDuration DurationJSON `json:"sumDuration,omitempty"`
	MinDuration DurationJSON `json:"minDuration,omitempty"`
	MaxDuration DurationJSON `json:"maxDuration,omitempty"`
}

type RestURL struct {
	sync.Mutex
	Description string         `json:"description,omitempty"`
	Method      string         `json:"method,omitempty"`
	Resource    string         `json:"resoure,omitempty"`
	Consumes    []string       `json:"consumes,omitempty"`
	Produces    []string       `json:"produces,omitempty"`
	Success     []int          `json:"success,omitempty"`
	Failure     []int          `json:"failure,omitempty"`
	Params      []RestURLField `json:"params,omitempty"`
	Stats       RestURLStats   `json:"stats,omitempty"`
	statsCh     chan time.Duration
}

func NewRestURL(method string, resource string) *RestURL {
	restURL := &RestURL{Method: method, Resource: resource, statsCh: make(chan time.Duration, 1000)}

	go restURL.updateStats()

	return restURL
}

func (restURL *RestURL) MuxString() string {
	return fmt.Sprintf("%s %s", restURL.Method, restURL.Resource)
}

func (restURL *RestURL) Format(args ...any) string {
	s := restURL.Resource

	if strings.Contains(s, "{") {
		regex := regexp.MustCompile("{.*?\\}")

		s = regex.ReplaceAllString(s, "%v")
		s = fmt.Sprintf(s, args...)
	}

	return s
}

func (restURL *RestURL) URL(address string, args ...any) string {
	return fmt.Sprintf("%s%s", address, restURL.Format(args...))
}

func (restURL *RestURL) Validate(r *http.Request) error {
	if restURL.Method != r.Method {
		return fmt.Errorf("invalid HTTP method")
	}

	rPaths := Split(r.URL.Path, "/")
	uPaths := Split(restURL.Resource, "/")

	if !strings.Contains(restURL.Resource, "*") {
		if len(rPaths) != len(uPaths) {
			return fmt.Errorf("invalid amount of HTTP request path")
		}
	}

	for _, param := range restURL.Params {
		if !r.URL.Query().Has(param.Name) && r.Header.Get(param.Name) == "" {
			if param.Default == "" && param.Mandatory {
				return fmt.Errorf("missing HTTP query/header param: %s", param.Name)
			}
		}
	}

	return nil
}

func (restURL *RestURL) ParamValue(r *http.Request, name string) string {
	v := r.URL.Query().Get(name)
	if v != "" {
		return v
	}

	v = r.Header.Get(name)
	if v != "" {
		return v
	}

	p := slices.IndexFunc(restURL.Params, func(field RestURLField) bool {
		return field.Name == name
	})

	if p != -1 {
		v = restURL.Params[p].Default
	}

	return v
}

func (restURL *RestURL) ParamValues(r *http.Request, name string) []string {
	list := []string{}

	v := r.URL.Query().Get(name)
	if v != "" {
		list = append(list, v)
	}

	v = r.Header.Get(name)
	if v != "" {
		list = append(list, r.Header.Values(name)...)
	}

	if len(list) > 0 {
		return list
	}

	p := slices.IndexFunc(restURL.Params, func(field RestURLField) bool {
		return field.Name == name
	})

	if p != -1 {
		v = restURL.Params[p].Default
	}

	return []string{v}
}

func (restURL *RestURL) updateStats() {
	for d := range restURL.statsCh {
		restURL.Lock()

		restURL.Stats.Count++
		restURL.Stats.SumDuration.Duration += d

		if restURL.Stats.Count == 1 {
			restURL.Stats.MinDuration.Duration = d
			restURL.Stats.MaxDuration.Duration = d
		} else {
			restURL.Stats.MinDuration.Duration = Min(restURL.Stats.MinDuration.Duration, d)
			restURL.Stats.MaxDuration.Duration = Max(restURL.Stats.MaxDuration.Duration, d)
		}

		restURL.Unlock()
	}
}

func (restURL *RestURL) Statistics() RestURLStats {
	restURL.Lock()
	defer func() {
		restURL.Unlock()
	}()

	stats := restURL.Stats

	return stats
}

func (restURL *RestURL) UpdateStats(start time.Time) {
	restURL.statsCh <- time.Since(start)
}

func (restURL *RestURL) SwaggerInfo() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("@Title %s\n", restURL.Resource))
	sb.WriteString(fmt.Sprintf("@Description %s\n", restURL.Description))
	sb.WriteString(fmt.Sprintf("@Accept %s\n", strings.Join(restURL.Consumes, " ")))

	return sb.String()
}
