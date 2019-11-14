package common

type resourceloader func(name string) []byte

var resourceloaders = make([]resourceloader, 0)

func RegisterResourceLoader(r resourceloader) {
	resourceloaders = append(resourceloaders, r)
}

func GetResource(name string) []byte {
	for _, resourceloader := range resourceloaders {
		ba := resourceloader(name)

		if ba != nil {
			return ba
		}
	}

	return nil
}
