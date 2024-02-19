package common

import (
	"fmt"
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
	}

	return resourcesDirectory
}

func ReadResource(filename string) ([]byte, string, error) {
	if app == nil || app.Resources == nil {
		return nil, "", fmt.Errorf("resources are not initialized")
	}

	ba, err := app.Resources.ReadFile(filename)
	if err != nil {
		ba, err = app.Resources.ReadFile(fmt.Sprintf("%s/%s", ResourcesDirectory(), filename))

		if err != nil {
			return nil, "", err
		}
	}

	resourcesLock.Lock()
	defer resourcesLock.Unlock()

	mimeType, ok := resourcesMimeTypes[filename]
	if !ok {
		mt, err := DetectMimeType(filename, ba)
		if err != nil {
			return nil, "", err
		}

		mimeType = mt.MimeType

		resourcesMimeTypes[filename] = mimeType
	}

	return ba, mimeType, nil
}
