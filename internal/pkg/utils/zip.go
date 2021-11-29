package utils

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ZipDir(dir string) ([]byte, error) {
	// create a new zip archive
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	logger.Debugf("creating zip archive...")
	zipWriter := zip.NewWriter(writer)

	// change to the directory
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	if err = os.Chdir(filepath.Dir(absPath)); err != nil {
		return nil, err
	}

	basePath := filepath.Base(absPath)
	err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		logger.Infof("write %s to archive...", path)
		if err != nil {
			return err
		}

		// create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(basePath), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		}

		// create writer for the file header and save content of the file
		headerWriter, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)

		return err
	})
	if err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("zipWriter.Close failed: %s", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("writer.Flush failed: %s", err)
	}

	return buf.Bytes(), nil
}

func Unzip(zipContent []byte, dstPath string) error {
	reader := bytes.NewReader(zipContent)
	zipReader, err := zip.NewReader(reader, int64(len(zipContent)))
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dstPath, 0700); err != nil {
		return err
	}

	extractAndWriterFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				logger.Errorf("rc.Close failed: %s", err)
				panic(err)
			}
		}()

		path := filepath.Join(dstPath, f.Name)
		mode := f.Mode()

		logger.Debugf("extracting %s...", path)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, mode); err != nil {
				return err
			}
		} else {

			// create a file to write the contents of the zip file
			if err := os.MkdirAll(filepath.Dir(path), mode); err != nil {
				return err
			}
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			defer func() {
				if err := file.Close(); err != nil {
					logger.Errorf("file.Close failed: %s", err)
					panic(err)
				}
			}()

			if _, err := io.Copy(file, rc); err != nil {
				return err
			}
		}

		return nil
	}

	for _, f := range zipReader.File {
		if err := extractAndWriterFile(f); err != nil {
			return err
		}
	}

	return nil
}
