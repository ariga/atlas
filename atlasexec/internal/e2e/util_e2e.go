package e2etest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"ariga.io/atlas/atlasexec"
)

const testFixtureDir = "testdata"

func runTestWithVersions(t *testing.T, versions []string, fixtureName string, cb func(t *testing.T, ver *atlasexec.Version, wd *atlasexec.WorkingDir, tf *atlasexec.Client)) {
	if os.Getenv("ATLASEXEC_E2ETEST") == "" {
		t.Skip("ATLASEXEC_E2ETEST not set")
	}
	t.Helper()
	alreadyRunVersions := map[string]bool{}
	for _, av := range versions {
		t.Run(fmt.Sprintf("%s-%s", fixtureName, av), func(t *testing.T) {
			if alreadyRunVersions[av] {
				t.Skipf("already run version %q", av)
			}
			alreadyRunVersions[av] = true

			var execPath string
			if localBinPath := os.Getenv("ATLASEXEC_E2ETEST_ATLAS_PATH"); localBinPath != "" {
				execPath = localBinPath
			} else {
				execPath = downloadAtlas(t, av)
				if err := os.Chmod(execPath, 0755); err != nil {
					t.Fatalf("unable to make atlas executable: %s", err)
				}
			}
			c, err := atlasexec.NewClient("", execPath)
			if err != nil {
				t.Fatal(err)
			}
			// TODO: Check that the version is the same as the one we expect.
			runningVersion, err := c.Version(context.Background())
			if err != nil {
				t.Fatalf("unable to determine running version (expected %q): %s", av, err)
			}
			wd, err := atlasexec.NewWorkingDir()
			if err != nil {
				t.Fatal(err)
			}
			defer wd.Close()
			if fixtureName != "" {
				err := wd.CopyFS("", os.DirFS(filepath.Join(testFixtureDir, fixtureName)))
				if err != nil {
					t.Fatalf("error copying config file into test dir: %s", err)
				}
			}
			err = c.WithWorkDir(wd.Path(), func(c *atlasexec.Client) (err error) {
				defer func() {
					if r := recover(); r != nil {
						var ok bool
						if err, ok = r.(error); !ok {
							err = fmt.Errorf("run test failure: %v", r)
						}
					}
				}()
				cb(t, runningVersion, wd, c)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func downloadAtlas(t *testing.T, version string) string {
	t.Helper()
	c := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, "https://atlasgo.sh?test=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "AtlasExec/Integration-Test")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", res.StatusCode)
	}
	path := filepath.Join(t.TempDir(), "installer.sh")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(f, res.Body); err != nil {
		f.Close()
		t.Fatal(err)
	} else if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	atlasBin := filepath.Join(t.TempDir(), "atlas")
	cmd := exec.Command(path,
		"--user-agent", "AtlasExec/Integration-Test",
		"--output", atlasBin,
		"--no-install",
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("ATLAS_VERSION=%s", version))
	if testing.Verbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err = cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return atlasBin
}
