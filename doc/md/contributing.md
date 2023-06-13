---
title: Contributing
id: contributing
slug: /contributing
---

### How to Contribute
Atlas is a community project, we welcome contributions of all kinds and sizes!

Here are some ways in which you can help:
* File well-written and descriptive bug reports or feature requests in the [Issues page](https://github.com/ariga/atlas/issues).
* Tweet about your experience with Atlas on [Twitter](https://twitter.com), don't forget to mention
  [@ariga_io](https://twitter.com/ariga_io) and link to [atlasgo.io](https://atlasgo.io) if you do.
* Write educational content on your personal blog or websites such as [dev.to](https://dev.to) or 
  [Medium](https://medium.com). If you do, don't hesitate to reach out to us via Discord (link below)
  for help proof-reading your text and 
  using our social-media channels for distributing it to readers.
* Join our [Discord Server](https://discord.com/invite/QhsmBAWzrC) to answer questions of other users
  and find out other ways in which you can contribute by talking to the community there!
* Contribute bug-fixes or new features to the [codebase](https://github.com/ariga/atlas).

### Contributing code to Atlas

As we are still starting out, we don't have an official code-style or guidelines on composing your
code. As general advice, read through the area of the code that you are modifying and try to keep your code
similar to what others have written in the same place.  

#### Code-generation

Some of the code in the Atlas repository is generated. The CI process verifies that
all generated files are checked-in by running `go generate ./...` and then running
`git status --porcelain`. Therefore, before committing changes to Atlas, please run:
```shell
go generate ./...
```

#### Linting

Your code will be linted using `golangci-lint` during CI. To install in locally,
[follow this guide](https://golangci-lint.run/usage/install/#local-installation). 

To run it locally:
```shell
golangci-lint run
```

#### Formatting 
Format your code using the standard `fmt` command:
```shell
go fmt ./...
```

#### Unit-tests

Your code should be covered in unit-tests, see the codebase for examples. To run tests:
```shell
go test ./...
```

#### Integration tests

Some features, especially those that interact directly with a database must be verified
in an integration test. There is extensive infrastructure for integration tests under
`internal/integration/` that runs tests under a matrix of database dialect (Postres, MySQL, etc.)
and versions. To run the integration tests, first use the `docker-compose.yml` file to spin up
databases to test against:

```shell
cd internal/integration 
docker-compose up -d
```

Then run the tests, from with the `integration` directory:
```shell
go test ./...
```

### Contributing documentation 

The Atlas documentation website is generated from the project's main [GitHub repo](https://github.com/ariga/atlas).

Follow this short guide to contribute documentation improvements and additions:

#### Setting Up

1. [Locally fork and clone](https://docs.github.com/en/github/getting-started-with-github/quickstart/fork-a-repo) the
  [repository](https://github.com/ariga/atlas).
2. The documentation site uses [Docusaurus](https://docusaurus.io/). To run it you will need [Node.js installed](https://nodejs.org/en/).
3. Install the dependencies:
  ```shell
  cd doc/website && npm install
  ```
4. Run the website in development mode:
  ```shell
  cd doc/website && npm start
  ```
5. Open you browser at [http://localhost:3000](http://localhost:3000).

#### General Guidelines

* Documentation files are located in `doc/md`, they are [Markdown-formatted](https://en.wikipedia.org/wiki/Markdown)
  with "front-matter" style annotations at the top. [Read more](https://docusaurus.io/docs/docs-introduction) about
  Docusaurus's document format.
* Atlas uses [Golang CommitMessage](https://github.com/golang/go/wiki/CommitMessage) formats to keep the repository's
  history nice and readable. As such, please use a commit message such as:
```text
doc/md: adding a guide on contribution of docs to atlas
```

#### Adding New Documents

1. Add a new Markdown file in the `doc/md` directory, for example `doc/md/writing-docs.md`.

2. The file should be formatted as such:
  ```markdown
  ---
  id: writing-docs
  title: Writing Docs
  ---
  ...
  ```
  Where `id` should be a unique identifier for the document,  and should be the same as the filename without the `.md` suffix,
  and `title` is the title of the document as it will appear in the page itself and any navigation element on the site.
3. If you want the page to appear in the documentation website's sidebar, add a `doc` block to `website/sidebars.js`, for example:
```diff
  {
    type: 'doc',
    id: 'writing-docs',
  },
+  {
+    type: 'doc',
+    id: 'contributing',
+  },
```