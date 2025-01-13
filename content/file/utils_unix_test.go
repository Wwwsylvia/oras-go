//go:build !windows

/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package file

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func Test_extractTarDirectory(t *testing.T) {
	tests := []struct {
		name      string
		tarData   []byte
		wantFiles map[string]string // map of file paths to their expected contents
		wantErr   bool
	}{
		{
			name: "extract valid files",
			tarData: createTar(t, []tarEntry{
				{name: "base/", mode: os.ModeDir | 0777},
				{name: "base/test.txt", content: "hello world", mode: 0666},
				{name: "base/file_symlink", linkname: "test.txt", mode: os.ModeSymlink | 0666},
				{name: "base/file_hardlink", linkname: "test.txt", mode: 0666, isHardLink: true},
			}),
			wantFiles: map[string]string{
				"base/test.txt":      "hello world",
				"base/file_symlink":  "hello world",
				"base/file_hardlink": "hello world",
			},
			wantErr: false,
		},
		{
			name: "non-regular files",
			tarData: createTar(t, []tarEntry{
				{name: "something", isNonRegular: true},
			}),
			wantErr: true,
		},
		{
			name:    "invalid tar header",
			tarData: []byte("random data"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			dirName := "base"
			dirPath := filepath.Join(tempDir, dirName)
			buf := make([]byte, 1024)

			if err := extractTarDirectory(dirPath, dirName, bytes.NewReader(tt.tarData), buf); (err != nil) != tt.wantErr {
				t.Fatalf("extractTarDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				for path, wantContent := range tt.wantFiles {
					filePath := filepath.Join(tempDir, path)
					fi, err := os.Lstat(filePath)
					if err != nil {
						t.Fatalf("failed to stat file %s: %v", filePath, err)
					}

					if fi.Mode()&os.ModeSymlink != 0 {
						filePath, err = os.Readlink(filePath)
						if err != nil {
							t.Fatalf("failed to read link %s: %v", filePath, err)
						}
						if !filepath.IsAbs(filePath) {
							filePath = filepath.Join(tempDir, filePath)
						}
					}
					gotContent, err := os.ReadFile(filePath)
					if err != nil {
						t.Fatalf("failed to read file %s: %v", filePath, err)
					}
					if string(gotContent) != wantContent {
						t.Errorf("file content = %s, want %s", gotContent, wantContent)
					}
				}
			}
		})
	}
}

type tarEntry struct {
	name         string
	content      string
	linkname     string
	mode         os.FileMode
	isNonRegular bool
	isHardLink   bool
}

func createTar(t *testing.T, entries []tarEntry) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, entry := range entries {
		hdr := &tar.Header{
			Name: entry.name,
			Mode: int64(entry.mode.Perm()),
			Size: int64(len(entry.content)),
		}
		if entry.isNonRegular {
			hdr.Typeflag = tar.TypeBlock
		} else if entry.isHardLink {
			hdr.Typeflag = tar.TypeLink
			hdr.Linkname = entry.linkname
		} else if entry.mode&os.ModeSymlink != 0 {
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = entry.linkname
		}

		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(entry.content)); err != nil {
			t.Fatalf("failed to write tar content: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	return buf.Bytes()
}
