package common

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"time"
)

type dirTime struct {
	name string
	ti   time.Time
}

func Unzip(dest, src string) error {
	Debug("unzip source: %s dest: %s", src, dest)

	r, err := zip.OpenReader(src)
	if Error(err) {
		return err
	}
	defer r.Close()

	_, local := time.Now().Zone()

	var dt []dirTime

	for _, zipEntry := range r.File {
		rc, err := zipEntry.Open()
		if Error(err) {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, zipEntry.Name)

		ti := zipEntry.Modified
		_, zipEntryUTC := ti.Zone()

		ti = ti.Add(time.Duration(zipEntryUTC-local) * time.Second)

		if zipEntry.FileInfo().IsDir() {
			err := os.RemoveAll(path)
			if Error(err) {
				return err
			}

			err = os.MkdirAll(path, zipEntry.Mode())
			if Error(err) {
				return err
			}

			dt = append(dt, dirTime{path, ti})
		} else {
			if FileExists(path) {
				err = os.Remove(path)
				if Error(err) {
					return err
				}
			}

			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipEntry.Mode())
			if Error(err) {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if Error(err) {
				return err
			}

			err = f.Close()
			if Error(err) {
				return err
			}

			err = os.Chtimes(path, ti, ti)
			if Error(err) {
				return err
			}
		}
	}

	for _, dt := range dt {
		err = os.Chtimes(dt.name, dt.ti, dt.ti)
		if Error(err) {
			return err
		}
	}

	return nil
}

func addFile(w *zip.Writer, fn, baseInZip string) error {
	Debug("addFile: %s", fn)

	if len(baseInZip) > 0 && baseInZip[len(baseInZip)-1:] != "/" {
		baseInZip += "/"
	}

	f, err := os.Open(fn)
	if Error(err) {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if Error(err) {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if Error(err) {
		return err
	}

	header.Name = baseInZip + filepath.Base(fn)
	header.Method = zip.Deflate
	header.Modified = info.ModTime()

	writer, err := w.CreateHeader(header)
	if Error(err) {
		return err
	}

	_, err = io.Copy(writer, f)
	if Error(err) {
		return err
	}

	return nil
}

func addFiles(w *zip.Writer, dir, baseInZip string) error {
	Debug("addFiles: %s", dir)

	if len(baseInZip) > 0 && baseInZip[len(baseInZip)-1:] != "/" {
		baseInZip += "/"
	}

	files, err := os.ReadDir(dir)
	if Error(err) {
		return err
	}

	for _, file := range files {
		fn := filepath.Clean(dir + string(filepath.Separator) + file.Name())

		if !file.IsDir() {
			err := addFile(w, fn, baseInZip)
			if Error(err) {
				return err
			}
		} else if file.IsDir() {
			err = addFiles(w, fn, baseInZip+filepath.Base(file.Name()))
			if Error(err) {
				return err
			}
		}
	}

	return nil
}

type ZipTarget struct {
	BaseDir string
	Files   []string
}

func Zip(filename string, targets []ZipTarget) error {
	outFile, err := os.Create(filename)
	if Error(err) {
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	for _, target := range targets {
		for _, file := range target.Files {
			fi, err := os.Stat(file)
			if Error(err) {
				return err
			}

			if fi.IsDir() {
				err = addFiles(w, file, target.BaseDir)
				if Error(err) {
					return err
				}
			} else {
				err = addFile(w, file, target.BaseDir)
				if Error(err) {
					return err
				}
			}
		}
	}

	return nil
}
