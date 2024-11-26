package common

import (
	"net/url"
	"strings"
)

type URI struct {
	Scheme   string
	Opaque   string
	Username string
	Password string
	Host     string
	Port     string
	Path     string
	Params   url.Values
}

const (
	SchemeSep       = "://"
	OpaqueSep       = ":"
	CredentialSep   = "@"
	CredentialSplit = ":"
	CidrSep         = "/"
	CidrSplit       = ":"
	CidrSplitV6     = "]:"
	ParamsSep       = "?"
	ParamSep        = "&"
	ParamValueSep   = "="
)

func NewURI(s string) (*URI, error) {
	uri := &URI{Params: make(url.Values)}

	p := strings.Index(s, SchemeSep)
	if p != -1 {
		uri.Scheme = s[:p]
		s = s[p+len(SchemeSep):]
	} else {
		p := strings.Index(s, OpaqueSep)
		if p != -1 {
			uri.Opaque = s[:p]
			uri.Path = s[p+len(OpaqueSep):]

			return uri, nil
		}
	}

	p = strings.Index(s, CredentialSep)
	if p != -1 {
		sub := s[:p]
		splits := strings.Split(sub, CredentialSplit)
		uri.Username = splits[0]
		if len(splits) > 1 {
			uri.Password = splits[1]
		}

		s = s[p+len(CredentialSep):]
	}

	var cidr string

	p = strings.Index(s, CidrSep)
	if p != -1 {
		cidr = s[:p]
		uri.Path = s[p:]
	} else {
		cidr = s
	}

	sep := CidrSplit
	sepAdd := 0
	if isV6(cidr) {
		sep = CidrSplitV6
		sepAdd = 1
	}

	p = strings.LastIndex(cidr, sep)
	if p != -1 {
		uri.Host = cidr[:p+sepAdd]
		uri.Port = cidr[p+len(sep):]
	} else {
		uri.Host = cidr
	}

	if strings.HasPrefix(uri.Scheme, "http") {
		p = strings.Index(uri.Path, ParamsSep)
		if p != -1 {
			paramsSplits := Split(uri.Path[p+1:], ParamSep)
			for _, paramsSplit := range paramsSplits {
				splits := Split(paramsSplit, ParamValueSep)

				value := ""
				if len(splits) > 1 {
					var err error

					value, err = url.QueryUnescape(splits[1])
					if Error(err) {
						return nil, err
					}
				}

				uri.Params.Add(splits[0], value)
			}

			uri.Path = uri.Path[:p]
		}
	}

	return uri, nil
}

func isV6(s string) bool {
	return strings.HasPrefix(s, "[")
}

func (uri *URI) IsV6() bool {
	return isV6(uri.Host)
}

func (uri *URI) String() string {
	sb := strings.Builder{}

	if uri.Opaque != "" {
		sb.WriteString(uri.Opaque)

		if uri.Path != "" {
			sb.WriteString(OpaqueSep)
			sb.WriteString(uri.Path)
		}

		return sb.String()
	}

	if uri.Scheme != "" {
		sb.WriteString(uri.Scheme)
		sb.WriteString(SchemeSep)
	}

	if uri.Username != "" || uri.Password != "" {
		if uri.Username != "" {
			sb.WriteString(uri.Username)
		}

		if uri.Password != "" {
			sb.WriteString(CredentialSplit)
			sb.WriteString(uri.Password)
		}

		sb.WriteString(CredentialSep)
	}

	if uri.Host != "" {
		sb.WriteString(uri.Host)
	}

	if uri.Port != "" {
		sb.WriteString(CidrSplit)
		sb.WriteString(uri.Port)
	}

	sb.WriteString(uri.Path)

	if len(uri.Params) > 0 {
		sb.WriteString("?")
		sb.WriteString(uri.Params.Encode())
	}

	return sb.String()
}

func (uri *URI) toURL() (*url.URL, error) {
	return url.Parse(uri.String())
}
