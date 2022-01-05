// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var inMemory = regexp.MustCompile("^file:.*:memory:$|:memory:|^file:.*mode=memory.*")

// SQLiteExists returns nil if the sqlite dsn is in memory or file exists, otherwise error.
func SQLiteExists(dsn string) error {
	if !inMemory.MatchString(dsn) {
		return fileExists(dsn)
	}
	return nil
}

func fileExists(dsn string) error {
	s := strings.Split(dsn, "?")
	f := dsn
	if len(s) == 2 {
		f = s[0]
	}
	if strings.Contains(f, "file:") {
		f = strings.SplitAfter(f, "file:")[1]
	}
	f = filepath.Clean(f)
	if _, err := os.Stat(f); err != nil {
		return fmt.Errorf("failed opening %q: %w", f, err)
	}
	return nil
}
