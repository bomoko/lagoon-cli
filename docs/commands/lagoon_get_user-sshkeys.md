## lagoon get user-sshkeys

Get a users SSH keys

### Synopsis

Get a users SSH keys. This will only work for users that are part of a group

```
lagoon get user-sshkeys [flags]
```

### Options

```
  -E, --email string   New email address of the user
  -h, --help           help for user-sshkeys
  -N, --name string    Name of the group to check users in (if not specified, will default to all groups)
```

### Options inherited from parent commands

```
      --all-projects         All projects (if supported)
      --debug                Enable debugging output (if supported)
  -e, --environment string   Specify an environment to use
      --force                Force (if supported)
  -l, --lagoon string        The Lagoon instance to interact with
      --no-header            No header on table (if supported)
      --output-csv           Output as CSV (if supported)
      --output-json          Output as JSON (if supported)
      --pretty               Make JSON pretty (if supported)
  -p, --project string       Specify a project to use
  -i, --ssh-key string       Specify a specific SSH key to use
      --version              Version information
```

### SEE ALSO

* [lagoon get](lagoon_get.md)	 - Get info on a project, or deployment
