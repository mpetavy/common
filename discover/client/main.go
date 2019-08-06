package main

import (
	"github.com/mpetavy/common"
	"github.com/mpetavy/common/discover"
)

func start() error {
	discoveredIps, err := discover.Discover("")
	if err != nil {
		return err
	}

	for k, v := range discoveredIps {
		common.Debug("discoveredIp %s: %s", k, v)
	}

	return nil
}

func main() {
	defer common.Cleanup()

	common.New(&common.App{common.Title(), "1.0.0", "2019", "discover demo client", "mpetavy", common.APACHE, "https://github.com/mpetavy/" + common.Title(), true, nil, start, nil, nil, 0}, nil)
	common.Run()
}
