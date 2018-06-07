package builder

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

type Artifact struct {
	writer *zip.Writer
}

func NewArtifact(filePath string) Artifact {
	zipFile, err := os.Create(filePath)
	errPanic(err)

	artifactWriter := zip.NewWriter(zipFile)
	return Artifact{
		writer: artifactWriter,
	}
}

func (a Artifact) Add(file string, archiveDir string) error {

	err := filepath.Walk(file, func(filePath string, fileInfo os.FileInfo, err error) error {
		archivePath := archiveDir + filepath.Base(filePath)
		if err != nil || fileInfo.IsDir() {
			return err
		}
		if err != nil {
			return err
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer func() {
			_ = file.Close()
		}()

		zipFileWriter, err := a.writer.Create(archivePath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipFileWriter, file)
		return err
	})

	return err
}

func (a Artifact) Flush() error {
	return a.writer.Close()
}
