package e2etest

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"ariga.io/atlas/atlasexec"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func Test_SQLite(t *testing.T) {
	runTestWithVersions(t, []string{"latest"}, "versioned-basic", func(t *testing.T, ver *atlasexec.Version, wd *atlasexec.WorkingDir, c *atlasexec.Client) {
		url := "sqlite://file.db?_fk=1"
		ctx := context.Background()
		s, err := c.MigrateStatus(ctx, &atlasexec.MigrateStatusParams{
			URL: url,
			Env: "local",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(s.Pending))
		require.Equal(t, "20240112070806", s.Pending[0].Version)

		r, err := c.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
			URL: url,
			Env: "local",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(r.Applied), "Should be one migration applied")
		require.Equal(t, "20240112070806", r.Applied[0].Version, "Should be the correct migration applied")

		// Apply again, should be a no-op.
		r, err = c.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
			URL: url,
			Env: "local",
		})
		require.NoError(t, err, "Should be no error")
		require.Equal(t, 0, len(r.Applied), "Should be no migrations applied")
	})
}

func Test_PostgreSQL(t *testing.T) {
	u := os.Getenv("ATLASEXEC_E2ETEST_POSTGRES_URL")
	if u == "" {
		t.Skip("ATLASEXEC_E2ETEST_POSTGRES_URL not set")
	}
	runTestWithVersions(t, []string{"latest"}, "versioned-basic", func(t *testing.T, ver *atlasexec.Version, wd *atlasexec.WorkingDir, c *atlasexec.Client) {
		url := u
		ctx := context.Background()
		s, err := c.MigrateStatus(ctx, &atlasexec.MigrateStatusParams{
			URL: url,
			Env: "local",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(s.Pending))
		require.Equal(t, "20240112070806", s.Pending[0].Version)

		r, err := c.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
			URL: url,
			Env: "local",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(r.Applied), "Should be one migration applied")
		require.Equal(t, "20240112070806", r.Applied[0].Version, "Should be the correct migration applied")

		// Apply again, should be a no-op.
		r, err = c.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
			URL: url,
			Env: "local",
		})
		require.NoError(t, err, "Should be no error")
		require.Equal(t, 0, len(r.Applied), "Should be no migrations applied")
	})
}

func Test_MultiTenants(t *testing.T) {
	t.Setenv("ATLASEXEC_E2ETEST_ATLAS_PATH", "atlas")
	runTestWithVersions(t, []string{"latest"}, "multi-tenants", func(t *testing.T, ver *atlasexec.Version, wd *atlasexec.WorkingDir, c *atlasexec.Client) {
		ctx := context.Background()
		r, err := c.MigrateApplySlice(ctx, &atlasexec.MigrateApplyParams{
			Env:    "local",
			Amount: 1, // Only apply one migration.
		})
		require.NoError(t, err)
		require.Len(t, r, 2, "Should be two tenants")
		require.Equal(t, 1, len(r[0].Applied), "Should be one migration applied")
		require.Equal(t, "20240112070806", r[0].Applied[0].Version, "Should be the correct migration applied")

		// Insert some data to the second tenant to make the migration fail.
		db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_fk=1", wd.Path("foo.db")))
		if err != nil {
			log.Fatalf("failed opening db: %s", err)
		}
		_, err = db.Exec("INSERT INTO t1(c1) VALUES (1),(1),(1)")
		require.NoError(t, err)

		// Apply again, should be one successful and one failed migration.
		_, err = c.MigrateApplySlice(ctx, &atlasexec.MigrateApplyParams{
			Env: "local",
		})
		require.ErrorContains(t, err, "UNIQUE constraint failed", "Should be error")
		mae, ok := err.(*atlasexec.MigrateApplyError)
		require.True(t, ok, "Should be a MigrateApplyError")
		require.Len(t, mae.Result, 2, "Should be two reports")
		require.Equal(t, 1, len(mae.Result[0].Applied), "Should be one migration applied")
		require.Equal(t, "20240116003831", mae.Result[0].Applied[0].Version, "Should be the correct migration applied")

		require.Equal(t, 1, len(mae.Result[1].Applied), "Should be one migration applied")
		require.Contains(t, mae.Result[1].Error, "UNIQUE constraint failed", "Should be the correct error")

		// Apply again, should be one successful and one failed migration.
		_, err = c.MigrateApplySlice(ctx, &atlasexec.MigrateApplyParams{
			Env: "local",
		})
		require.ErrorContains(t, err, "UNIQUE constraint failed", "Should be error")
		mae, ok = err.(*atlasexec.MigrateApplyError)
		require.True(t, ok, "Should be a MigrateApplyError")
		require.Len(t, mae.Result, 2, "Should be two reports")

		require.Equal(t, 0, len(mae.Result[0].Applied), "Should be no migrations applied")
		require.Equal(t, 1, len(mae.Result[1].Applied), "Should be one tried to apply")
		require.Contains(t, mae.Result[1].Error, "UNIQUE constraint failed", "Should be the correct error")
	})
}

func Test_SchemaPlan(t *testing.T) {
	runTestWithVersions(t, []string{"latest"}, "schema-plan", func(t *testing.T, ver *atlasexec.Version, wd *atlasexec.WorkingDir, c *atlasexec.Client) {
		ctx := context.Background()
		plan, err := c.SchemaPlan(ctx, &atlasexec.SchemaPlanParams{
			From:   []string{"file://schema-1.lt.hcl"},
			To:     []string{"file://schema-2.lt.hcl"},
			DevURL: "sqlite://:memory:?_fk=1",
			DryRun: true,
		})
		require.NoError(t, err)
		f := plan.File
		require.NotNil(t, f, "Should have a file")
		require.Equal(t, "-- Add column \"c2\" to table: \"t1\"\nALTER TABLE `t1` ADD COLUMN `c2` text NOT NULL;\n", f.Migration, "Should be the correct migration")
		require.Empty(t, f.URL, "Should be no URL")
	})
	runTestWithVersions(t, []string{"latest"}, "schema-plan", func(t *testing.T, ver *atlasexec.Version, wd *atlasexec.WorkingDir, c *atlasexec.Client) {
		ctx := context.Background()
		plan, err := c.SchemaPlan(ctx, &atlasexec.SchemaPlanParams{
			From:   []string{"file://schema-1.lt.hcl"},
			To:     []string{"file://schema-2.lt.hcl"},
			DevURL: "sqlite://:memory:?_fk=1",
			Save:   true,
		})
		require.NoError(t, err)
		f := plan.File
		require.NotNil(t, f, "Should have a file")
		require.Equal(t, "-- Add column \"c2\" to table: \"t1\"\nALTER TABLE `t1` ADD COLUMN `c2` text NOT NULL;\n", f.Migration, "Should be the correct migration")
		require.Regexp(t, "^file://\\d{14}\\.plan\\.hcl$", f.URL, "Should be an URL to a file")
	})
}
