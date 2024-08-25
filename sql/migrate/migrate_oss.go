// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package migrate

import (
	"context"
	"fmt"
)

// IsCheckpoint reports whether the file is a checkpoint file.
func (f *LocalFile) IsCheckpoint() bool {
	return f.isCheckpoint()
}

// CheckpointTag returns the tag of the checkpoint file, if defined.
func (f *LocalFile) CheckpointTag() (string, error) {
	return f.checkpointTag()
}

// fileStmts returns the statements defined in the given file.
func (e *Executor) fileStmts(f File) ([]*Stmt, error) {
	return FileStmtDecls(e.drv, f)
}

func (e *Executor) fileChecks(context.Context, File, *Revision) error {
	return nil // unimplemented
}

// ValidateDir before operating on it.
func (e *Executor) ValidateDir(context.Context) error {
	if err := Validate(e.dir); err != nil {
		return fmt.Errorf("sql/migrate: validate migration directory: %w", err)
	}
	return nil
}
