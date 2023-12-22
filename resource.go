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
	if resourcesDirectory == "" {
		entries, _ := app.Resources.ReadDir(".")

		for _, entry := range entries {
			if entry.IsDir() {
				resourcesDirectory = entry.Name()

				return resourcesDirectory
			}
		}

		Panic(fmt.Errorf("cannot find resource directory"))
	}

	return resourcesDirectory
}

func ReadResource(filename string) ([]byte, string, error) {
	if app == nil || app.Resources == nil {
		return nil, "", fmt.Errorf("resources are not initialized")
	}

	path := filename
	if !strings.HasPrefix(path, ResourcesDirectory()) {
		path = strings.Join([]string{ResourcesDirectory(), filename}, "/")
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
