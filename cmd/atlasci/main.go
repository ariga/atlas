// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"ariga.io/atlas/cmd/atlasci/internal/ci"
	_ "ariga.io/atlas/cmd/atlascmd/docker"
	"ariga.io/atlas/sql/migrate"
	_ "ariga.io/atlas/sql/mysql"
	_ "ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var opt options
	check(opt.parse())
	ctx := context.Background()
	dev, err := sqlclient.Open(ctx, opt.devURL)
	check(err)
	dir, err := migrate.NewLocalDir(opt.dir)
	check(err)
	format := ci.DefaultTemplate
	if opt.format != "" {
		format, err = template.New("format").
			Funcs(ci.TemplateFuncs).
			Parse(opt.format)
		check(err)
	}
	detect := ci.LatestChanges(dir, opt.detectFrom.latest)
	if opt.detectFrom.gitBase != "" {
		detect, err = ci.NewGitChangeDetector(
			opt.detectFrom.gitRoot, dir,
			ci.WithBase(opt.detectFrom.gitBase),
			ci.WithMigrationsPath(opt.dir),
		)
		check(err)
	}
	r := &ci.Runner{
		Dev:            dev,
		Scan:           dir,
		ChangeDetector: detect,
		ReportWriter: &ci.TemplateWriter{
			T: format,
			W: os.Stdout,
		},
		Analyzer: sqlcheck.Analyzers{
			// Add more analyzers here.
			sqlcheck.Destructive,
		},
	}
	check(r.Run(ctx))
}

type options struct {
	dir        string // migration dir.
	format     string // custom log format.
	devURL     string // dev database url.
	detectFrom struct {
		latest  int    // latest N migration files.
		gitBase string // git branch name.
		gitRoot string // repository root.
	}
}

var usage = `Usage:  atlasci [options]

Options:
  --dir          string   Select migration directory
  --format       string   Custom logging using a Go template
  --dev-url      string   Select a data source using the URL format
  --latest       int      Run analysis on the latest N migration files
  --git-base     string   Run analysis against the base Git branch
  --git-root     string   Path to the repository root

Additional commands:
  license        Display license information for this program.

Examples:
  atlasci --dir path/to/dir --dev-url mysql://root:pass@localhost:3306 --latest 1
  atlasci --dir path/to/dir --dev-url mysql://root:pass@localhost:3306 --git-base master
  atlasci --dir path/to/dir --dev-url mysql://root:pass@localhost:3306 --latest 1 --format '{{ json .Files }}'
`

func (o *options) parse() error {
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.StringVar(&o.dir, "dir", os.Getenv("ATLASCI_DIR"), "")
	flag.StringVar(&o.devURL, "dev-url", os.Getenv("ATLASCI_DEV_URL"), "")
	flag.StringVar(&o.format, "format", "", "")
	flag.IntVar(&o.detectFrom.latest, "latest", 0, "")
	flag.StringVar(&o.detectFrom.gitBase, "git-base", os.Getenv("ATLASCI_GIT_BASE"), "")
	flag.StringVar(&o.detectFrom.gitRoot, "git-root", os.Getenv("ATLASCI_GIT_ROOT"), "")
	flag.Parse()
	if args := flag.Args(); len(args) > 0 && args[0] == "license" {
		fmt.Println(license)
		os.Exit(0)
	}
	var errors []string
	if o.dir == "" {
		errors = append(errors, "--dir is required")
	}
	if o.devURL == "" {
		errors = append(errors, "--dev-url is required")
	}
	if o.detectFrom.latest > 0 && o.detectFrom.gitBase != "" {
		errors = append(errors, "--latest and --git-base are mutually exclusive")
	}
	if o.detectFrom.latest == 0 && o.detectFrom.gitBase == "" {
		errors = append(errors, "--latest or --git-base is required")
	}
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ", "))
	}
	if o.detectFrom.gitRoot == "" {
		o.detectFrom.gitRoot = "."
	}
	return nil
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var license = `LICENSE

Atlas CI is licensed under Apache 2.0 as found in https://github.com/ariga/atlas/blob/master/LICENSE.`
