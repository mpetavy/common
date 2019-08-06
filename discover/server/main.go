package main

import (
	"github.com/mpetavy/common"
	"github.com/mpetavy/common/discover"
)

var (
	discoverServer discover.Server
)

func start() error {
	discoverServer = discover.NewServer("", 0, "", "")
	return discoverServer.Start()
}

func stop() error {
	return discoverServer.Stop()
}

func main() {
	defer common.Cleanup()

	common.New(&common.App{common.Title(), "1.0.0", "2019", "discover demo server", "mpetavy", common.APACHE, "https://github.com/mpetavy/" + common.Title(), true, nil, start, stop, nil, 0}, nil)
	common.Run()
}
