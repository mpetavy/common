package common

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	FlagIoFileResource *bool
	resourcesDirectory string
	resourcesMimeTypes = make(map[string]string)
)

const (
	FlagNameIoFileResource = "io.fileresource"
)

func init() {
	FlagIoFileResource = flag.Bool(FlagNameIoFileResource, !IsRunningAsExecutable(), "read resource from filesystem")
}

func ResourcesDirectory() string {
	return resourcesDirectory
}

func ReadResource(filename string) ([]byte, string, error) {
	var ba []byte
	var err error

	if app.Resources == nil {
		return nil,"", fmt.Errorf("resources are not initialized")
	}

	if resourcesDirectory == "" {
		de,_ := app.Resources.ReadDir(".")

		if len(de) == 1 {
			resourcesDirectory = de[0].Name()
		}
	}

	path := filename
	if !strings.HasPrefix(path, resourcesDirectory) {
		path = strings.Join([]string{resourcesDirectory, filename}, "/")
	}

	if *FlagIoFileResource {
		ba, _ = os.ReadFile(path)
	}

	if ba == nil {
		ba, err = app.Resources.ReadFile(path)
	}

	if Error(err) {
		return nil, "", err
	}

	mimeType, ok := resourcesMimeTypes[filename]
	if !ok {
		mt := DetectMimeType(filename, ba)
		mimeType = mt.MimeType

		resourcesMimeTypes[filename] = mimeType
	}

	return ba, mimeType, nil
}
