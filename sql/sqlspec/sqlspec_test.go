// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlspec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMightHeredoc(t *testing.T) {
	for _, tt := range []struct {
		input    string
		expected string
	}{
		{
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			input: `
SELECT
  *
  FROM users
  WHERE active`,
			expected: `<<-SQL
  SELECT
    *
    FROM users
    WHERE active
  SQL`,
		},
		{
			input: `
-- The line below includes spaces.
  
	
SELECT
  *
  FROM users
  WHERE active`,
			expected: `<<-SQL
  -- The line below includes spaces.


  SELECT
    *
    FROM users
    WHERE active
  SQL`,
		},
	} {
		require.Equal(t, tt.expected, MightHeredoc(tt.input))
	}
}
