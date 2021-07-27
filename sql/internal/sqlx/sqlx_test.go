package sqlx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionNames(t *testing.T) {
	names := VersionPermutations("mysql", "1.2.3")
	require.EqualValues(t, []string{"mysql", "mysql 1", "mysql 1.2", "mysql 1.2.3"}, names)

	names = VersionPermutations("postgres", "11.3 nightly")
	require.EqualValues(t, []string{"postgres", "postgres 11", "postgres 11.3", "postgres 11.3.nightly"}, names)
}
