package cuemod

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed gen module.cue
var cueModFS embed.FS

// Copy copies the contents of the embedded cue.mod directory to the given path.
func Copy(target string) (err error) {
	// ensurer directory is a cue.mod directory
	// maybe rethink this...
	if !strings.HasSuffix(target, "/cue.mod") {
		target = filepath.Join(target, "cue.mod")
	}

	if err = os.MkdirAll(target, 0755); err != nil {
		return err
	}

	matches, err := fs.Glob(cueModFS, "./*")
	if err != nil {
		return
	}

	for _, file := range matches {
		var stat fs.FileInfo
		stat, err = fs.Stat(cueModFS, file)
		if err != nil {
			return
		}

		if stat.IsDir() {
			err = copyDir(copyDirParams{
				fsRoot:    file,
				targetDir: target,
			})
		} else {
			err = copyFile(copyFileParams{
				embeddedFile: file,
				targetDir:    target,
			})
		}

		if err != nil {
			return
		}
	}

	return err
}

type copyDirParams struct {
	fsRoot    string
	targetDir string
}

func copyDir(params copyDirParams) error {
	return fs.WalkDir(cueModFS, params.fsRoot, func(path string, d fs.DirEntry, _ error) (err error) {
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(params.targetDir, path), 0755)
		}

		err = copyFile(copyFileParams{
			embeddedFile: path,
			targetDir:    params.targetDir,
		})
		if err != nil {
			return
		}
		return nil
	})
}

type copyFileParams struct {
	targetDir    string
	embeddedFile string
}

func copyFile(params copyFileParams) (err error) {
	vf, err := cueModFS.Open(params.embeddedFile)
	if err != nil {
		return err
	}

	dst := filepath.Join(params.targetDir, params.embeddedFile)
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, vf)
	if err != nil {
		return err
	}
	return nil
}
