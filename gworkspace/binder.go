package gworkspace

import (
	"github.com/nmcclain/ldap"
	"github.com/rs/zerolog/log"
	"net"
)

type WorkspaceBinder struct {
}

func (w *WorkspaceBinder) Bind(bindDN, bindSimplePw string, conn net.Conn) (ldap.LDAPResultCode, error) {
	log.Debug().Str("bindDN", bindDN).Msg("Incoming bind request")
	return ldap.LDAPResultSuccess, nil
}
