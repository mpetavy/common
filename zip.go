package common

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
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
	if err != nil {
		return err
	}
	defer r.Close()

	_, local := time.Now().Zone()

	var dt []dirTime

	for _, zipEntry := range r.File {
		rc, err := zipEntry.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, zipEntry.Name)

		ti := zipEntry.Modified
		_, zipEntryUTC := ti.Zone()

		ti = ti.Add(time.Duration(zipEntryUTC-local) * time.Second)

		if zipEntry.FileInfo().IsDir() {
			err := os.RemoveAll(path)
			if err != nil {
				return err
			}

			err = os.MkdirAll(path, zipEntry.Mode())
			if err != nil {
				return err
			}

			dt = append(dt, dirTime{path, ti})
		} else {
			b, err := FileExists(path)
			if err != nil {
				return err
			}

			if b {
				err = os.Remove(path)
				if err != nil {
					return err
				}
			}

			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipEntry.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}

			f.Close()

			err = os.Chtimes(path, ti, ti)
			if err != nil {
				return err
			}
		}
	}

	for _, dt := range dt {
		err = os.Chtimes(dt.name, dt.ti, dt.ti)
		if err != nil {
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
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = baseInZip + filepath.Base(fn)
	header.Method = zip.Deflate
	header.Modified = info.ModTime()

	writer, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, f)
	if err != nil {
		return err
	}

	return nil
}

func addFiles(w *zip.Writer, dir, baseInZip string) error {
	Debug("addFiles: %s", dir)

	if len(baseInZip) > 0 && baseInZip[len(baseInZip)-1:] != "/" {
		baseInZip += "/"
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		fn := filepath.Clean(dir + string(filepath.Separator) + file.Name())

		if !file.IsDir() {
			err := addFile(w, fn, baseInZip)
			if err != nil {
				return err
			}
		} else if file.IsDir() {
			err = addFiles(w, fn, baseInZip+filepath.Base(file.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Zip(filename string, files []string) error {

	outFile, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	for _, file := range files {
		fi, err := os.Stat(file)
		if err != nil {
			break
		}

		if fi.IsDir() {
			err = addFiles(w, file, "")
		} else {
			err = addFile(w, file, "")
		}
		if err != nil {
			break
		}
	}

	return nil
}
