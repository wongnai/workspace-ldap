# LDAP Bridge for Google Workspace

This service implement an LDAP server using user and group information from Google Workspace Admin API.

The server is intended to be used as a [group mapping info provider](https://docs.paloaltonetworks.com/pan-os/8-1/pan-os-admin/user-id/map-users-to-groups.html) for Palo Alto Networks firewalls.

## Setup

1. Set `GOOGLE_APPLICATION_CREDENTIALS=/path/to/serviceaccount.json` (see next section)
2. Run Docker with `--impersonate domain-admin@example.com --base-dn example.com`

## Service account
If using service account for authentication, make sure it is configured for [Domain-wide delegation](https://developers.google.com/admin-sdk/directory/v1/guides/delegation).

Scopes needed

- https://www.googleapis.com/auth/admin.directory.user.readonly
- https://www.googleapis.com/auth/admin.directory.group.readonly
- https://www.googleapis.com/auth/admin.directory.group.member.readonly

## Directory layout
See [docs](docs/tree.md)

## Caveats
- This dump the entire Google directory (users/groups) into memory, so it would take long time to start
- `memberOf` on user is not implemented
- Binds is not implemented. Any bind on the base DN would return success
- SASL is not implemented in the upstream library. Don't send SASL request to this server!
- This is NOT a drop in replacement for [Secure LDAP service](https://support.google.com/a/answer/9048516?hl=en)

## License

[Apache License 2.0](LICENSE)
