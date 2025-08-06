# Atlas SDK for Go

[![Go Reference](https://pkg.go.dev/badge/ariga.io/atlas-go-sdk/atlasexec.svg)](https://pkg.go.dev/ariga.io/atlas@master/atlasexec)

An SDK for building ariga/atlas providers in Go.

## Installation

```bash
go get -u ariga.io/atlas@master
```

## How to use

To use the SDK, you need to create a new client with your `migrations` folder and the `atlas` binary path.

```go
package main

import (
    ...
    "ariga.io/atlas/atlasexec"
)

func main() {
    // Create a new client
    client, err := atlasexec.NewClient("my-migration-folder", "my-atlas-cli-path")
    if err != nil {
        log.Fatalf("failed to create client: %v", err)
    }
}
```

## APIs

For more information, refer to the documentation available at [GoDoc](https://pkg.go.dev/ariga.io/atlas@master/atlasexec#Client)