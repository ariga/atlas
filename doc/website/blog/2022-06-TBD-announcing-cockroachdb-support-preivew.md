---
title: "Announcing the preview support for CockroachDB"
authors: Hedwigz
tags: [cockroach, cockroachdb, integration, announcement]
image: https://blog.ariga.io/uploads/images/posts/cockroachdb/cockroachdb.png
---

Today we announce the official preview support for CockroachDB. Both for [Ent](https://entgo.io) and [Atlas](https://atlasgo.io).

# Atlas
[Atlas](https://atlasgo.io) is an open source tool that helps developers manage their database schemas. It has a [CLI tool](https://atlasgo.io/cli/reference) and [Terraform integration](https://atlasgo.io/blog/2022/05/04/announcing-terraform-provider). Lately, Atlas became the [migration Engine for Ent](https://entgo.io/blog/2022/01/20/announcing-new-migration-engine).

# CockroachDB
[CockroachDB](https://www.cockroachlabs.com/) (CRDB) is an [open-source](https://github.com/cockroachdb/cockroach) NewSQL database. From their README:
> CockroachDB is a distributed SQL database built on a transactional and strongly-consistent key-value store. It scales horizontally; survives disk, machine, rack, and even datacenter failures with minimal latency disruption and no manual intervention; supports strongly-consistent ACID transactions; and provides a familiar SQL API for structuring, manipulating, and querying data.  
  
CockroachDB has been gaining popularity and many of you [have](https://github.com/ent/ent/issues/2545) [been](https://github.com/ariga/atlas/issues/785#issue-1231951038) [asking](https://github.com/ariga/atlas/issues/785#issuecomment-1125853135) us about it.

While CockroachDB aims to be PostgreSQL compatible, it still has some incompatibilities (e.g. [1](https://github.com/cockroachdb/cockroach/issues/20296#issuecomment-1066140651),[2](https://github.com/cockroachdb/cockroach/issues/82064),[3](https://github.com/cockroachdb/cockroach/issues/81659)) which prevent the naive postgres dialect from working with it. Due to this, we had to work around these issues.  
  
With the latest release of Atlas, the postgres dialect automatically detects when it is connected to CRDB and uses a custom Driver to handle that.

### Preview support
The integration of Atlas with CRDB is well tested with CRDB version `v21.2.11` (at the time of writing, `latest`) and we will extend that in the future.
If you're using other versions of CRDB or looking for help, don't hesitate to [file an issue](https://github.com/ariga/atlas/issues) or join our [Discord channel](https://discord.gg/zZ6sWVg6NT).

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
