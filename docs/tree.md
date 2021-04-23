# LDAP Tree
## cn=groups
Store Google groups

LDAP Attribute     | Google Attribute        | Description
-------------------|-------------------------|---------------
objectclass        |                         | Always "group"
cn                 | Email                   | The group's email address
mail               | Email                   | The group's email address
mail               | Aliases                 | List of a group's alias email addresses.
member             |                         | List of group members
uid                | Id                      | The unique ID of a group

## cn=org
Store OU (Not implemented)

## cn=users
Store users

LDAP Attribute     | Google Attribute        | Description
-------------------|-------------------------|---------------
objectclass        |                         | Always "user"
cn                 | PrimaryEmail            | Primary username
givenname          | Name.GivenName          | First Name
surname            | Name.FamilyName         | Last name
mail               | Aliases                 | Email
uid                | Id                      | User ID
has2Fa             | IsEnrolledIn2Sv         | Is enrolled in 2-step verification
archived           | Archived                | Indicates if user is archived
suspended          | Suspended               | Indicates if user is suspended
