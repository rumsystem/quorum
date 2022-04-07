package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ZipDir zip files in a directory, do not include the directory itself
func ZipDir(dir string, zipPath string) error {
	logger.Infof("creating zip archive for %s => %s ...", dir, zipPath)

	// create a new zip archive
	outZipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer outZipFile.Close()

	zipWriter := zip.NewWriter(outZipFile)
	defer zipWriter.Close()

	// do not change working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory failed: %s", err)
	}
	defer os.Chdir(currentDir)

	// change to the directory
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	if err = os.Chdir(absPath); err != nil {
		return err
	}

	basePath := "." // current directory
	err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		logger.Debugf("write %s to archive...", path)
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
		return err
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("zipWriter.Close failed: %s", err)
	}

	return nil
}

func Unzip(zipPath string, dstPath string) error {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

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
