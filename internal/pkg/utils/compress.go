package utils

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

// Compress compress with zstd
func Compress(in io.Reader, out io.Writer) error {
	enc, err := zstd.NewWriter(out)
	if err != nil {
		return err
	}

	_, err = io.Copy(enc, in)
	if err != nil {
		enc.Close()
		return err
	}

	return enc.Close()
}

// Decompress decompress with zstd
func Decompress(in io.Reader, out io.Writer) error {
	d, err := zstd.NewReader(in)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(out, d)

	return err
}
