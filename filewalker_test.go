package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	ROOT_DIR  = "filewalker_test"
	ROOT_FILE = "root-file"
	SUB_DIR   = "sub"
	SUB_FILE  = "sub-file"
)

func createFiles(filePrefix string) error {
	for i := 0; i < 3; i++ {
		f, err := os.Create(fmt.Sprintf("%s.%d", filePrefix, i))
		if Error(err) {
			return err
		}
		err = f.Close()
		if Error(err) {
			return err
		}
	}

	return nil
}

func removeTestFolders(root string) error {
	if FileExists(root) {
		err := os.RemoveAll(root)
		if Error(err) {
			return err
		}
	}

	return nil
}

func createTestFolders() (string, string, error) {
	root := filepath.Join(os.TempDir(), ROOT_DIR)
	sub := filepath.Join(root, SUB_DIR)

	err := removeTestFolders(root)
	if Error(err) {
		return "", "", err
	}

	err = os.MkdirAll(sub, DefaultDirMode)
	if Error(err) {
		return "", "", err
	}

	err = createFiles(filepath.Join(root, ROOT_FILE))
	if Error(err) {
		return "", "", err
	}

	err = createFiles(filepath.Join(sub, SUB_FILE))
	if Error(err) {
		return "", "", err
	}

	return root, sub, nil
}

func TestFilewalker(t *testing.T) {
	InitTesting(t)

	root, sub, err := createTestFolders()
	if Error(err) {
		return
	}

	defer func() {
		Error(removeTestFolders(root))
	}()

	type fields struct {
		Path        string
		Filemask    string
		Recursive   bool
		IgnoreError bool
		walkFunc    func(path string, f os.FileInfo) error
	}
	founds := []string{}
	foundFunc := func(path string, f os.FileInfo) error {
		founds = append(founds, path)

		sort.Strings(founds)

		return nil
	}
	tests := []struct {
		name      string
		fields    fields
		wantFiles []string
	}{
		{
			"0",
			fields{
				Filemask:    root,
				Recursive:   false,
				IgnoreError: false,
				walkFunc:    foundFunc,
			},
			[]string{
				root,
				filepath.Join(root, ROOT_FILE+".0"),
				filepath.Join(root, ROOT_FILE+".1"),
				filepath.Join(root, ROOT_FILE+".2"),
			},
		},
		{
			"1",
			fields{
				Filemask:    filepath.Join(root, "*.1"),
				Recursive:   false,
				IgnoreError: false,
				walkFunc:    foundFunc,
			},
			[]string{
				root,
				filepath.Join(root, ROOT_FILE+".1"),
			},
		},
		{
			"2",
			fields{
				Filemask:    root,
				Recursive:   true,
				IgnoreError: false,
				walkFunc:    foundFunc,
			},
			[]string{
				root,
				filepath.Join(root, ROOT_FILE+".0"),
				filepath.Join(root, ROOT_FILE+".1"),
				filepath.Join(root, ROOT_FILE+".2"),
				sub,
				filepath.Join(sub, SUB_FILE+".0"),
				filepath.Join(sub, SUB_FILE+".1"),
				filepath.Join(sub, SUB_FILE+".2"),
			},
		},
		{
			"3",
			fields{
				Filemask:    filepath.Join(root, "*.1"),
				Recursive:   true,
				IgnoreError: false,
				walkFunc:    foundFunc,
			},
			[]string{
				root,
				filepath.Join(root, ROOT_FILE+".1"),
				sub,
				filepath.Join(sub, SUB_FILE+".1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			founds = []string{}
			err := WalkFiles(
				tt.fields.Filemask,
				tt.fields.Recursive,
				tt.fields.IgnoreError,
				tt.fields.walkFunc,
			)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.wantFiles), len(founds))
			for _, item := range tt.wantFiles {
				assert.True(t, SliceContains(founds, item))
			}
		})
	}
}
