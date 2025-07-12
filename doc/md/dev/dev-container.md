---
id: dev-container
title: Manipulating Development Containers
---

Atlas creates and manages Docker containers for development databases. Sometimes, you may need to interact with these containers directly, such as:

- Creating tablespaces for PostgreSQL
- Installing database extensions
- Copying configuration files into the container
- Setting up specific directory structures

The `atlas dev container` command group provides functionality to interact with development database containers.

## Prerequisites

- Docker must be installed and running on your machine
- The container must be running and accessible

## Finding Your Container ID

To interact with a container, you need its ID or name. You can find this using standard Docker commands:
