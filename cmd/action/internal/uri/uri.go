// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package uri

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var inMemory = regexp.MustCompile("^file:.*:memory:$|:memory:|^file:.*mode=memory.*")

// SqliteExists returns nil if the sqlite dsn is in memory or file exists, otherwise error.
func SqliteExists(dsn string) error {
	if !inMemory.MatchString(dsn) {
		return fileExists(dsn)
	}
	return nil
}

func fileExists(dsn string) error {
	s := strings.Split(dsn, "?")
	fn := dsn
	if len(s) == 2 {
		fn = s[0]
	}
	if strings.Contains(fn, "file:") {
		fn = strings.SplitAfter(fn, "file:")[1]
	}
	fn = filepath.Clean(fn)
	_, err := os.Stat(fn)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file %s does not exist", fn)
	}
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("no permission to access file %s", fn)
	}
	return nil
}
