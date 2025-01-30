package common

import (
	"flag"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	uid  = "discover"
	info = "info"
)

var (
	address string
)

func TestMain(m *testing.M) {
	flag.Parse()
	Exit(m.Run())
}

func discoverServer(backgroundTask *BackgroundTask) {
	server, err := NewDiscoverServer(address, time.Second, uid, info)
	if Error(err) {
		return
	}

	err = server.Start()
	if Error(err) {
		return
	}

	<-backgroundTask.Channel()

	err = server.Stop()
	if Error(err) {
		return
	}
}

func TestDiscover(t *testing.T) {
	port, err := FindFreePort("tcp", 2048, nil)
	require.NoError(t, err)

	address = fmt.Sprintf(":%d", port)

	list, err := Discover(address, time.Second, uid)
	require.NoError(t, err)
	require.Equal(t, 0, len(list))

	btServer := NewBackgroundTask(discoverServer)

	btServer.Start()

	list, err = Discover(address, time.Second, uid)
	require.NoError(t, err)

	require.Equal(t, 1, len(list))
	require.Equal(t, list[0], info)

	defer btServer.Stop(true)
}
