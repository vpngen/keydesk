package kdlib

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
)

func PemGzipBase64(b *pem.Block) ([]byte, error) {
	buf := new(bytes.Buffer)

	b64w := base64.NewEncoder(base64.StdEncoding.WithPadding(base64.StdPadding), buf)
	gzw := gzip.NewWriter(b64w)

	err := pem.Encode(gzw, b)
	if err != nil {
		return nil, fmt.Errorf("pem encode: %w", err)
	}

	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}

	if err := b64w.Close(); err != nil {
		return nil, fmt.Errorf("base64 close: %w", err)
	}

	return buf.Bytes(), nil
}

func PemGzip(b *pem.Block) ([]byte, error) {
	buf := new(bytes.Buffer)

	gzw := gzip.NewWriter(buf)

	err := pem.Encode(gzw, b)
	if err != nil {
		return nil, fmt.Errorf("pem encode: %w", err)
	}

	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}

	return buf.Bytes(), nil
}

func GzipBase64(b []byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	b64w := base64.NewEncoder(base64.StdEncoding.WithPadding(base64.StdPadding), buf)

	gzw := gzip.NewWriter(b64w)
	if _, err := io.Copy(gzw, bytes.NewReader(b)); err != nil {
		return nil, fmt.Errorf("copy: %w", err)
	}

	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}

	if err := b64w.Close(); err != nil {
		return nil, fmt.Errorf("base64 close: %w", err)
	}

	return buf.Bytes(), nil
}

func Unbase64Ungzip(s string) ([]byte, error) {
	b64r := base64.NewDecoder(base64.StdEncoding.WithPadding(base64.StdPadding), bytes.NewBufferString(s))

	gzr, err := gzip.NewReader(b64r)
	if err != nil {
		return nil, fmt.Errorf("gzip new reader: %w", err)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, gzr); err != nil {
		return nil, fmt.Errorf("copy: %w", err)
	}

	if err := gzr.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}

	return buf.Bytes(), nil
}

func Gzip(b []byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	gzw := gzip.NewWriter(buf)
	if _, err := io.Copy(gzw, bytes.NewReader(b)); err != nil {
		return nil, fmt.Errorf("copy: %w", err)
	}

	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}

	return buf.Bytes(), nil
}

func Ungzip(b []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("gzip new reader: %w", err)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, gzr); err != nil {
		return nil, fmt.Errorf("copy: %w", err)
	}

	if err := gzr.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}

	return buf.Bytes(), nil
}
