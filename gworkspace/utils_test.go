package gworkspace

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStrBool(t *testing.T) {
	assert.Equal(t, "true", boolStr(true))
	assert.Equal(t, "false", boolStr(false))
}

func TestFqdnToLdap(t *testing.T) {
	assert.Equal(t, "", FqdnToLdap("", "cn"))
	assert.Equal(t, "dc=lmwn,dc=com", FqdnToLdap("lmwn.com", "dc"))
	assert.Equal(t, "cn=test,cn=example,cn=com", FqdnToLdap("test.example.com", "cn"))
}
