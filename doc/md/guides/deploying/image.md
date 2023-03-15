---
id: image
slug: /guides/deploying/image
title: Creating container images for migrations
---

To integrate schema migrations into pipelines that deploy to container management
systems (such as Kubernetes, AWS ECS, Google Cloud Run, etc.) it is recommended
to create a dedicated container image per version that contains the
migration tool (such as Atlas) and the relevant migration files.

In this guide we will demonstrate how to build a dedicated Docker image that
includes Atlas and the relevant migrations files. We will demonstrate
how to build this image as a [GitHub Actions Workflow](https://docs.github.com/en/actions/using-workflows),
but the same result can be achieved in any CI system. 

## Defining the Dockerfile

Suppose our project structure looks something like: 

```text
.
├── main.go
└── migrations
    ├── 20221031125934_init.sql
    ├── 20221031125940_add_users_table.sql
    ├── 20221031125948_add_products_table.sql
    └── atlas.sum
```

Our goal is to build an image that contains:
1. The `migrations` directory
2. The Atlas binary

To do this we can build our container image with the official [Atlas Docker image](https://hub.docker.com/r/arigaio/atlas)
as [the base layer](https://docs.docker.com/engine/reference/builder/#from).  

To do this, our Dockerfile should be placed in the directory _containing_ the `migrations`
directory and will look something like this:

```dockerfile title=Dockerfile
FROM arigaio/atlas:latest

COPY migrations /migrations
```

## Verify our image

To test our new Dockerfile run:

```text
docker build -t my-image .
```

Docker will build our image:

```text
 => [internal] load build definition from Dockerfile                                          0.0s
 => => transferring dockerfile: 36B                                                           0.0s
 => [internal] load .dockerignore                                                             0.0s
 => => transferring context: 2B                                                               0.0s
 => [internal] load metadata for docker.io/arigaio/atlas:latest                               0.0s
 => [internal] load build context                                                             0.0s
 => => transferring context: 252B                                                             0.0s
 => [1/2] FROM docker.io/arigaio/atlas:latest                                                 0.0s
 => CACHED [2/2] COPY migrations /migrations                                                  0.0s
 => exporting to image                                                                        0.0s
 => => exporting layers                                                                       0.0s
 => => writing image sha256:c928104de31fc4c99d114d40ea849ade917beae3df7ffe9326113b289939878e  0.0s
 => => naming to docker.io/library/my-image                                                   0.0s
```

To verify Atlas can find your migrations directory and that its [integrity](/concepts/migration-directory-integrity)
is intact run:

```text
docker run --rm my-image migrate validate 
```

If no issues are found, no errors will be printed out.

## Defining the GitHub Actions Workflow

Next, we define a GitHub Actions workflow that will build our container
image and push it to the GitHub container repo (ghcr.io) on every push
to our mainline branch:

```yaml title=.github/workflows/push-docker.yaml
name: Push Docker
on:
  push:
    branches:
      - master
jobs:
  docker-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      - name: Login to GitHub Container Registry
        uses: docker/login-action@v2
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push
        uses: docker/build-push-action@v3
        with:
          push: true
          file: ${{ matrix.file }}
          tags: ghcr.io/ariga/<repo name>:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

Save this file in your GitHub repository under the `.github/workflows` directory.
After you push it to your mainline branch, you will see a run of the new
workflow in the [Actions](https://docs.github.com/en/actions/monitoring-and-troubleshooting-workflows/viewing-workflow-run-history)
tab of the repository.

