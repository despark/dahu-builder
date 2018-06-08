package builder

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

type Artifact struct {
	writer *zip.Writer
	dirs   []string
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
	err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if archiveDir != "" {
			header.Name = archiveDir + filepath.Base(path)

			if archiveDir[len(archiveDir)-1:] != "/" {
				header.Method = zip.Deflate
			} else {
				// we need to create the dir in the archive if it's missing
				dirHeader, err := func(name string) (*zip.FileHeader, error) {
					fh := &zip.FileHeader{
						Name:               name,
						UncompressedSize64: uint64(0),
					}
					fh.SetMode(os.ModeDir)

					return fh, nil
				}(archiveDir)

				_, err = a.writer.CreateHeader(dirHeader)

				if err != nil {
					return err
				}

				if err != nil {
					return err
				}
			}
		}

		writer, err := a.writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func (a Artifact) Flush() error {
	return a.writer.Close()
}
