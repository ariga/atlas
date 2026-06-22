## Usage

Reference the feature in your `devcontainer.json`:

```jsonc
{
    "features": {
        "ghcr.io/ariga/atlas/cli:1": {
            "version": "latest"
        }
    }
}
```

To pin a specific Atlas CLI version, set the `version` option (without the leading `v`):

```jsonc
{
    "features": {
        "ghcr.io/ariga/atlas/cli:1": {
            "version": "1.1.0"
        }
    }
}
```

## Extended build / flavors

Set the `flavor` option to install the extended Atlas build with a beta-only
driver. Supported values: `oracle`, `snowflake`, `spanner`, `databricks`.

```jsonc
{
    "features": {
        "ghcr.io/ariga/atlas/cli:1": {
            "version": "latest",
            "flavor": "oracle"
        }
    }
}
```

Leave `flavor` empty (the default) to install the standard build.

## Included VS Code Extensions

When used in a VS Code / GitHub Codespaces devcontainer, this feature
automatically installs the [Atlas HCL](https://marketplace.visualstudio.com/items?itemName=Ariga.atlas-hcl)
extension (`Ariga.atlas-hcl`) for syntax highlighting and language support
of Atlas HCL schema files.

## OS Support

This feature relies on the official Atlas install script (https://atlasgo.sh) and
supports Debian/Ubuntu based devcontainers. `curl` and `ca-certificates` are
installed automatically when missing.
