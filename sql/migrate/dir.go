// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"io"
	"regexp"
)

// SplitBaseline splits the directory files into two groups.
// The first one holds all baseline files and the second
// holds all file that need to be executed in the migration.
func SplitBaseline(dir Dir) ([]File, []File, error) {
	files, err := dir.Files()
	if err != nil {
		return nil, nil, err
	}
	idx := -1
	for i, f := range files {
		if f.Baseline() {
			idx = i
		}
	}
	if idx == -1 {
		return nil, files, nil
	}
	// In case baseline files were found, skip all
	// files prior to the latest one including it.
	return files[:idx+1], files[idx+1:], nil
}

const (
	directiveSum  = "sum"
	sumModeIgnore = "ignore"
	// atlas:baseline directive.
	directiveBaseline = "baseline"
	// atlas:delimiter directive.
	directiveDelimiter = "delimiter"
	directivePrefixSQL = "-- "
)

var reDirective = regexp.MustCompile(`^([ -~]*)atlas:(\w+)(?: +([ -~]*))*`)

// Directive searches in the content a line that matches a directive
// with the given prefix and name. For example:
//
//	Directive(b, "-- ", "delimiter")
//	Directive(b, "", "sum")
//
func Directive(content, prefix, name string) (string, bool) {
	m := reDirective.FindStringSubmatch(content)
	if len(m) == 4 && m[1] == prefix && m[2] == name {
		return m[3], true
	}
	return "", false
}

// readHashFile reads the HashFile from the given Dir.
func readHashFile(dir Dir) (HashFile, error) {
	f, err := dir.Open(HashFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var fh HashFile
	if err := fh.UnmarshalText(b); err != nil {
		return nil, err
	}
	return fh, nil
}
