// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysqlversion_test

import (
	"testing"

	"ariga.io/atlas/sql/mysql/internal/mysqlversion"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestV_SupportsGeneratedColumns(t *testing.T) {
	tests := []struct {
		v    string
		want bool
	}{
		{"5.6", false},
		{"5.7", true},
		{"5.7.0", true},
		{"5.7.40-0ubuntu0.18.04.1", true},
		{"8.0.0", true},
		{"10.1.1-MariaDB", false},
		{"10.2.1-MariaDB-10.2.1+maria~bionic", true},
		{"10.3.1-MariaDB-10.2.1+maria~bionic-log", true},
	}
	for _, tt := range tests {
		t.Run(tt.v, func(t *testing.T) {
			if got := mysqlversion.V(tt.v).SupportsGeneratedColumns(); got != tt.want {
				t.Errorf("V.SupportsGeneratedColumns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestV_CollateToCharset(t *testing.T) {
	c2c, err := mysqlversion.V("8.0.0").CollateToCharset(nil)
	require.NoError(t, err)
	require.Equal(t, "utf8mb4", c2c["utf8mb4_general_ci"])
	require.Empty(t, c2c["custom"])

	db, mk, err := sqlmock.New()
	require.NoError(t, err)
	mk.ExpectQuery("SELECT COLLATION_NAME, CHARACTER_SET_NAME FROM INFORMATION_SCHEMA.COLLATIONS").
		WillReturnRows(sqlmock.NewRows([]string{"COLLATION_NAME", "CHARACTER_SET_NAME"}).AddRow("custom", "unknown"))
	c2c, err = mysqlversion.V("8.0.0").CollateToCharset(db)
	require.NoError(t, err)
	require.Equal(t, "unknown", c2c["custom"])
}
