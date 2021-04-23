package gworkspace

import (
	"github.com/nmcclain/ldap"
	"strings"
)

func boolStr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func FqdnToLdap(fqdn string, t string) string {
	if fqdn == "" {
		return ""
	}

	parts := strings.Split(fqdn, ".")
	out := strings.Builder{}

	for _, part := range parts {
		out.WriteString(t)
		out.WriteRune('=')
		out.WriteString(part)
		out.WriteRune(',')
	}

	return out.String()[:out.Len()-1]
}

func CloneLdapEntry(input *ldap.Entry) (out ldap.Entry) {
	out.DN = input.DN
	out.Attributes = make([]*ldap.EntryAttribute, len(input.Attributes))
	for i, value := range input.Attributes {
		cloned := CloneLdapAttribute(value)
		out.Attributes[i] = &cloned
	}
	return
}

func CloneLdapAttribute(input *ldap.EntryAttribute) (out ldap.EntryAttribute) {
	out.Name = input.Name
	out.Values = make([]string, len(input.Values))
	copy(out.Values, input.Values)
	return
}
