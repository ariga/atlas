---
id: Welcome to Atlas
slug: /CLI/
sidebar_position: 2
---

## CLI

Work with any data source from the command line.

![Alt Text](https://release.ariga.io/images/assets/atlas-intro.gif)

## Installation

Download from one of our links or using curl.
```shell
curl -LO https://release.ariga.io/atlas/atlas-{os}-{arch}-{version}
```
Make the atlas binary executable.
```shell
chmod +x ./atlas-{os}-{arch}-{version}
```
Move the atlas binary to a file location on your system PATH.

Example for Mac:
```shell
sudo mv ./atlas-darwin-{arch}-{version} /usr/local/bin/atlas
sudo chown root: /usr/local/bin/atlas
```
Example for Linux:
```shell
sudo install -o root -g root -m 0755 ./atlas-linux-{arch}-{version} /usr/local/bin/atlas
```

|Latest Release                        |
|--------------------------------|
| [atlas-darwin-amd64-v0.0.1](https://release.ariga.io/atlas/atlas-darwin-amd64-v0.0.1)            |
| [atlas-darwin-arm64-v0.0.1](https://release.ariga.io/atlas/atlas-darwin-arm64-v0.0.1)          |
| [atlas-linux-amd64-v0.0.1](https://release.ariga.io/atlas/atlas-linux-amd64-v0.0.1)          |
| [atlas-windows-amd64-v0.0.1.exe](https://release.ariga.io/atlas/atlas-windows-amd64-v0.0.1.exe)          |
