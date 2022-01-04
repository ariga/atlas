## atlas env

Print atlas env params

### Synopsis

Env prints atlas environment information.
Every set environment param will print in the form of NAME=VALUE.

List of supported environment parameters:
"ATLAS_NO_UPDATE_NOTIFIER": On any command, the CLI will check for updates with the GitHub public API once every 24 hours.
To cancel this behavior, set the environment parameter "ATLAS_NO_UPDATE_NOTIFIER".

```
atlas env [flags]
```

### Options

```
  -h, --help   help for env
```

### SEE ALSO

* [atlas](atlas.md)	 - A database toolkit.

