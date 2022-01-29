---
title: Contributing
id: contributing
slug: /contributing
---
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