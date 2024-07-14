package common

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type RestURLField struct {
	Name        string
	Description string
	Default     string
	Optional    bool
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
	PathValues  []RestURLField
	PathParams  []RestURLField
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

	for _, param := range u.PathParams {
		if !r.URL.Query().Has(param.Name) {
			if param.Optional {
				if param.Default != "" {
					r.URL.Query().Add(param.Name, param.Default)
				}
			} else {
				return fmt.Errorf("missing HTTP header: %s", param.Name)
			}
		}
	}

	for _, header := range u.Headers {
		if r.Header.Get(header.Name) == "" {
			if header.Optional {
				if header.Default != "" {
					r.Header.Add(header.Name, header.Default)
				}
			} else {
				return fmt.Errorf("missing HTTP header: %s", header.Name)
			}
		}
	}

	return nil
}

func (u *RestURL) SwaggerInfo() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("@Title %s\n", u.Resource))
	sb.WriteString(fmt.Sprintf("@Description %s\n", u.Description))
	sb.WriteString(fmt.Sprintf("@Accept %s\n", strings.Join(u.Consumes, " ")))

	return sb.String()
}
