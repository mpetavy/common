package common

import (
	"fmt"
	"golang.org/x/exp/slices"
	"net/http"
	"regexp"
	"strings"
)

type RestURLField struct {
	Name        string
	Description string
	Default     string
}

type RestURL struct {
	Description string
	Method      string
	Resource    string
	Consumes    []string
	Produces    []string
	Success     []int
	Failure     []int
	Headers     []RestURLField
	Values      []RestURLField
}

func NewRestURL(method string, resource string) *RestURL {
	return &RestURL{Method: method, Resource: resource}
}

func (u *RestURL) MuxString() string {
	return fmt.Sprintf("%s %s", u.Method, u.Resource)
}

func (u *RestURL) Format(args ...any) string {
	s := u.Resource

	if strings.Contains(s, "{") {
		regex := regexp.MustCompile("{.*?\\}")

		s = regex.ReplaceAllString(s, "%v")
		s = fmt.Sprintf(s, args...)
	}

	return s
}

func (u *RestURL) URL(address string, args ...any) string {
	return fmt.Sprintf("%s%s", address, u.Format(args...))
}

func (u *RestURL) Validate(r *http.Request) error {
	if u.Method != r.Method {
		return fmt.Errorf("invalid HTTP method")
	}

	rPaths := Split(r.URL.Path, "/")
	uPaths := Split(u.Resource, "/")

	if !strings.Contains(u.Resource, "*") {
		if len(rPaths) != len(uPaths) {
			return fmt.Errorf("invalid amount of HTTP request path")
		}
	}

	for _, header := range u.Headers {
		if r.Header.Get(header.Name) == "" {
			if header.Default == "" {
				return fmt.Errorf("missing HTTP header: %s", header.Name)
			}
		}
	}

	for _, value := range u.Values {
		if !r.URL.Query().Has(value.Name) {
			if value.Default == "" {
				return fmt.Errorf("missing HTTP value: %s", value.Name)
			}
		}
	}

	return nil
}

func (u *RestURL) CleanHeader(r *http.Request, name string) string {
	v := r.Header.Get(name)
	if v == "" {
		p := slices.IndexFunc(u.Headers, func(field RestURLField) bool {
			return field.Name == name
		})

		if p != -1 {
			v = u.Headers[p].Default
		}
	}

	return v
}

func (u *RestURL) CleanValue(r *http.Request, name string) string {
	q := r.URL.Query()
	v := q.Get(name)
	if v == "" {
		p := slices.IndexFunc(u.Values, func(field RestURLField) bool {
			return field.Name == name
		})

		if p != -1 {
			v = u.Values[p].Default
		}
	}

	return v
}

func (u *RestURL) SwaggerInfo() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("@Title %s\n", u.Resource))
	sb.WriteString(fmt.Sprintf("@Description %s\n", u.Description))
	sb.WriteString(fmt.Sprintf("@Accept %s\n", strings.Join(u.Consumes, " ")))

	return sb.String()
}
