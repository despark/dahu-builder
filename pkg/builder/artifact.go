package builder

import (
	"fmt"
	copy2 "github.com/otiai10/copy"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Artifact struct {
	bin    string
	file   string
	tmpDir string
}

func NewArtifact(filePath string) Artifact {
	tmpDir, err := ioutil.TempDir("", "")
	errPanic(err)

	return Artifact{
		file:   filePath,
		bin:    findCommand("zip"),
		tmpDir: tmpDir,
	}
}

func (a Artifact) Add(file string, archiveDir string) error {
	err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {

		if err != nil || info.IsDir() {
			return err
		}

		if archiveDir != "" {
			if !strings.HasSuffix(archiveDir, "/") {
				archiveDir += "/"
			}
		}

		destDir := a.tmpDir + "/" + archiveDir
		_, err = os.Stat(destDir)

		if os.IsNotExist(err) {
			os.MkdirAll(destDir, os.FileMode(0755))
		} else {
			errPanic(err)
		}

		err = copy2.Copy(path, destDir+info.Name())

		return err
	})

	return err
}

func (a Artifact) Flush() error {
	fmt.Println(a.tmpDir)

	_exec(a.bin, a.tmpDir, "-r", a.file, ".")
	return nil
}
