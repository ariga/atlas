// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package main

//go:generate go run . -flavor Community -suffix oss

func init() {
	data.GoVersions = goVersions{"1.21"}
	data.Jobs = append(jobs,
		Job{
			Version: "tidb5",
			Image:   "pingcap/tidb:v5.4.0",
			Regex:   "TiDB",
			Ports:   []string{"4309:4000"},
		},
		Job{
			Version: "tidb6",
			Image:   "pingcap/tidb:v6.0.0",
			Regex:   "TiDB",
			Ports:   []string{"4310:4000"},
		},
		Job{
			Version: "cockroach-21",
			Image:   "ghcr.io/ariga/cockroachdb-single-node:v21.2.11",
			Regex:   "Cockroach",
			Ports:   []string{"26257:26257"},
		},
		Job{
			Version: "cockroach-22",
			Image:   "ghcr.io/ariga/cockroachdb-single-node:v22.1.0",
			Regex:   "Cockroach",
			Ports:   []string{"26258:26257"},
		},
		Job{
			Version: "cockroach-23",
			Image:   "ghcr.io/ariga/cockroachdb-single-node:v23.1.14",
			Regex:   "Cockroach",
			Ports:   []string{"26259:26257"},
		},
	)
}
