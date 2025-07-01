package atlasexec_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"ariga.io/atlas/atlasexec"
)

func ExampleClient_MigrateApply() {
	// Define the execution context, supplying a migration directory
	// and potentially an `atlas.hcl` configuration file using `atlasexec.WithHCL`.
	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(
			os.DirFS("./migrations"),
		),
	)
	if err != nil {
		log.Fatalf("failed to load working directory: %v", err)
	}
	// atlasexec works on a temporary directory, so we need to close it
	defer workdir.Close()

	// Initialize the client.
	client, err := atlasexec.NewClient(workdir.Path(), "atlas")
	if err != nil {
		log.Fatalf("failed to initialize client: %v", err)
	}
	// Run `atlas migrate apply` on a SQLite database under /tmp.
	res, err := client.MigrateApply(context.Background(), &atlasexec.MigrateApplyParams{
		URL: "sqlite:///tmp/demo.db?_fk=1&cache=shared",
	})
	if err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}
	fmt.Printf("Applied %d migrations\n", len(res.Applied))
}
