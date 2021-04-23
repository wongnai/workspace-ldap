package gworkspace

import (
	"github.com/nmcclain/ldap"
	"github.com/rs/zerolog"
)

type searchMarshaler struct {
	data *ldap.SearchRequest
}

func (s *searchMarshaler) MarshalZerologObject(e *zerolog.Event) {
	e.Str("baseDN", s.data.BaseDN)
	e.Str("filter", s.data.Filter)
	e.Int("sizeLimit", s.data.SizeLimit)
	e.Int("timeLimit", s.data.TimeLimit)
	e.Strs("attributes", s.data.Attributes)
}

func LogSearchObject(data *ldap.SearchRequest) zerolog.LogObjectMarshaler {
	return &searchMarshaler{data: data}
}
