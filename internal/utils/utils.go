package utils

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/psanford/memfs"
)

func Sha256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func RandomHash() string {
	data := make([]byte, 1024)
	rand.Read(data)
	return Sha256(string(data))
}

func Unzip(data []byte, fs *memfs.FS) error {
	reader := bytes.NewReader(data)
	r, _ := zip.NewReader(reader, reader.Size())

	for _, f := range r.File {
		fpath := filepath.Join("./", f.Name)
		fpath = strings.ReplaceAll(fpath, "\\", "/")
		if f.FileInfo().IsDir() {
			fs.MkdirAll(fpath, os.ModePerm)
			continue
		}

		fs.MkdirAll(filepath.Dir(fpath), os.ModePerm)

		rc, err := f.Open()
		if err != nil {
			return err
		}

		buffer := make([]byte, f.FileInfo().Size()+1024)
		n, _ := rc.Read(buffer)
		buffer = buffer[:n]
		fs.WriteFile(fpath, buffer, 0666)

		rc.Close()
	}
	return nil
}
