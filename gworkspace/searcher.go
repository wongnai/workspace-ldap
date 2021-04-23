package gworkspace

import (
	"context"
	"errors"
	"fmt"
	"github.com/nmcclain/ldap"
	"github.com/rs/zerolog/log"
	admin "google.golang.org/api/admin/directory/v1"
	"net"
	"strings"
	"sync"
)

const customerID = "my_customer"

type WorkspaceSearcher struct {
	MaxGroups int

	admin      *admin.Service
	baseDomain string

	cache []*ldap.Entry
	lock  sync.RWMutex
}

func NewSearcher(adm *admin.Service, baseDomain string) *WorkspaceSearcher {
	return &WorkspaceSearcher{
		admin:      adm,
		baseDomain: baseDomain,
	}
}

func (s *WorkspaceSearcher) Search(boundDN string, req ldap.SearchRequest, conn net.Conn) (ldap.ServerSearchResult, error) {
	log.Info().Str("boundDN", boundDN).Object("request", LogSearchObject(&req)).Msg("Incoming search request")

	if len(req.Attributes) == 1 && req.Attributes[0] == "supportedSASLMechanisms" {
		return s.saslRequest(boundDN, req, conn)
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	var out []*ldap.Entry
	for _, item := range s.cache {
		if item.DN == req.BaseDN || strings.HasSuffix(item.DN, ","+req.BaseDN) {
			cloned := CloneLdapEntry(item)
			out = append(out, &cloned)
		}
	}

	return ldap.ServerSearchResult{
		Entries:    out,
		ResultCode: ldap.LDAPResultSuccess,
	}, nil
}

func (s *WorkspaceSearcher) Update(ctx context.Context) error {
	users, err := s.dumpUsers(ctx)
	if err != nil {
		return err
	}
	groups, err := s.dumpGroups(ctx)
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.cache = append(users, groups...)
	log.Info().Msgf("%d entries fetched", len(s.cache))
	return nil
}

func (s *WorkspaceSearcher) dumpUsers(ctx context.Context) ([]*ldap.Entry, error) {
	var out []*ldap.Entry
	call := s.admin.Users.List().Customer(customerID).MaxResults(500)

	log.Debug().Msgf("Fetching users %+v", call)
	err := call.Pages(ctx, func(users *admin.Users) error {
		for _, user := range users.Users {
			out = append(out, s.convertUser(user))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *WorkspaceSearcher) convertUser(user *admin.User) *ldap.Entry {
	log.Trace().Str("primaryEmail", user.PrimaryEmail).Str("id", user.Id).Msg("Converting user")
	emails := []string{user.PrimaryEmail}
	emails = append(emails, user.Aliases...)

	attrs := []*ldap.EntryAttribute{
		{
			Name:   "objectclass",
			Values: []string{"user"},
		},
		{
			Name:   "cn",
			Values: []string{user.PrimaryEmail},
		},
		{
			Name:   "givenname",
			Values: []string{user.Name.GivenName},
		},
		{
			Name:   "surname",
			Values: []string{user.Name.FamilyName},
		},
		{
			Name:   "mail",
			Values: emails,
		},
		{
			Name:   "uid",
			Values: []string{user.Id},
		},
		{
			Name:   "has2Fa",
			Values: []string{boolStr(user.IsEnrolledIn2Sv)},
		},
		{
			Name:   "archived",
			Values: []string{boolStr(user.Archived)},
		},
		{
			Name:   "suspended",
			Values: []string{boolStr(user.Suspended)},
		},
	}

	return &ldap.Entry{
		DN:         s.userDn(user.PrimaryEmail),
		Attributes: attrs,
	}
}

func (s *WorkspaceSearcher) userDn(email string) string {
	return fmt.Sprintf("cn=%s,cn=users,%s", email, FqdnToLdap(s.baseDomain, "dc"))
}

func (s *WorkspaceSearcher) dumpGroups(ctx context.Context) ([]*ldap.Entry, error) {
	var out []*ldap.Entry
	stopIteration := errors.New("stop iteration")

	call := s.admin.Groups.List().Customer(customerID).MaxResults(200)

	log.Debug().Msgf("Fetching groups %+v", call)
	err := call.Pages(ctx, func(groups *admin.Groups) error {
		for _, group := range groups.Groups {
			out = append(out, s.convertGroup(ctx, group))

			if s.MaxGroups > 0 && len(out) >= s.MaxGroups {
				return stopIteration
			}
		}
		return nil
	})
	if err != nil && err != stopIteration {
		return nil, err
	}

	return out, nil
}

func (s *WorkspaceSearcher) convertGroup(ctx context.Context, group *admin.Group) *ldap.Entry {
	logger := log.With().Str("email", group.Email).Str("id", group.Id).Logger()

	logger.Debug().Msg("Reading group members")
	var members []string
	err := s.admin.Members.List(group.Id).Pages(ctx, func(mbrs *admin.Members) error {
		for _, user := range mbrs.Members {
			members = append(members, s.userDn(user.Email))
		}
		return nil
	})
	if err != nil {
		logger.Err(err).Msg("Fail to fetch group members")
		return nil
	}

	logger.Trace().Msg("Converting group")

	emails := []string{group.Email}
	emails = append(emails, group.Aliases...)

	attrs := []*ldap.EntryAttribute{
		{
			Name:   "objectclass",
			Values: []string{"group"},
		},
		{
			Name:   "cn",
			Values: []string{group.Email},
		},
		{
			Name:   "mail",
			Values: emails,
		},
		{
			Name:   "uid",
			Values: []string{group.Id},
		},
		{
			Name:   "member",
			Values: members,
		},
	}

	return &ldap.Entry{
		DN:         s.groupDn(group.Email),
		Attributes: attrs,
	}
}

func (s *WorkspaceSearcher) groupDn(email string) string {
	return fmt.Sprintf("cn=%s,cn=groups,%s", email, FqdnToLdap(s.baseDomain, "dc"))
}

func (s *WorkspaceSearcher) toLdapError(err error) ldap.LDAPResultCode {
	if errors.Is(err, context.DeadlineExceeded) {
		return ldap.LDAPResultTimeLimitExceeded
	}

	return ldap.LDAPResultOperationsError
}

func (s *WorkspaceSearcher) saslRequest(dn string, req ldap.SearchRequest, conn net.Conn) (ldap.ServerSearchResult, error) {
	return ldap.ServerSearchResult{
		Entries: []*ldap.Entry{
			{
				DN: "",
				Attributes: []*ldap.EntryAttribute{
					{
						Name:   "supportedSASLMechanisms",
						Values: []string{},
					},
				},
			},
		},
		ResultCode: ldap.LDAPResultSuccess,
	}, nil
}
