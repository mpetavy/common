package common

import (
	"fmt"
	"strings"
	"sync"
)

var (
	resourcesDirectory string
	resourcesMimeTypes = make(map[string]string)
	resourcesLock      sync.Mutex
)

func ResourcesDirectory() string {
	return resourcesDirectory
}

func ReadResource(filename string) ([]byte, string, error) {
	if app.Resources == nil {
		return nil, "", fmt.Errorf("resources are not initialized")
	}

	if resourcesDirectory == "" {
		de, _ := app.Resources.ReadDir(".")

		if len(de) == 1 {
			resourcesDirectory = de[0].Name()
		}
	}

	path := filename
	if !strings.HasPrefix(path, resourcesDirectory) {
		path = strings.Join([]string{resourcesDirectory, filename}, "/")
	}

	ba, err := app.Resources.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	resourcesLock.Lock()
	defer resourcesLock.Unlock()

	mimeType, ok := resourcesMimeTypes[filename]
	if !ok {
		mt := DetectMimeType(filename, ba)
		mimeType = mt.MimeType

		resourcesMimeTypes[filename] = mimeType
	}

	return ba, mimeType, nil
}
