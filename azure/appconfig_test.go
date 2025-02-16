package azure

import (
	"github.com/mpetavy/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAzureAppConfiguration(t *testing.T) {
	tenantID := "23feb136-a94f-44bb-b6ff-e9d9e598f33b" // petavy@gmx.net
	clientID := "0385068d-4874-4a8c-a6f3-c87d0acd73b7" // testapp
	clientSecret, err := common.Secret("secret:+j3WxWEJuWvrwNoGiMiWZEHoNJH6pMXzlVbotQx6XST++EltFWn8+A+5eHT0B8N5AegBVnr2gP4=")
	require.NoError(t, err)

	connection, err := common.Secret("secret:FXRnwuwS1v+ign7OTOH8usnYv5In+rPe8qQUlTvfsmobw2CEMTuuW6Wxk4aPgQiKRRXsRH5vBAy7AA8Z5rxTBeEIb0SzC4F/NjG3B3yKCZjWwUIzMluak/eWFMigxn3iOdMYfV5IF5ZVoxg3s1SbM289K6rPLix9OpU0fqCYvHZU0wcLkLCSxUaQP0dUC1ptvLdWm8zSPR/oQHJO+eQ=")
	require.NoError(t, err)

	// Test Sushant
	//tenantID := "82913d90-8716-4025-a8e8-4f8dfa42b719" // CZ MED150 - Connected-Customer-Care
	//clientID := "0385068d-4874-4a8c-a6f3-c87d0acd73b7" // dataingress-non-prod
	//clientSecret, err := common.Secret("secret:aC77Gt3KOulaAu67IOUws0U9RMQTkIGKlY+tE6GiftuFI4kw26GK/W7oRGmluDbPJKc1oImXRBU=")
	//require.NoError(t, err)
	//
	//connection, err := common.Secret("secret:biCnm/WwkmLYe1qKZgebP/uG1k/HY9QS6990h9hanKDxI9//kMA3OJsHdbWHAyVxa2KIBy/v4x65wST+xEgin2pMKKmq0SrUgiVkhA8afcc9XCeKUpwdyGvvlgd2L55hpKQ11Plff9RiZNQx/vP53YEUXQAQErt4+RJMrEBVOvhfWx+lOyWjQVip0NTux2NFwcgYxEVurVOILfATnLlW5iUXJxe9pJI=")
	//require.NoError(t, err)

	flags, err := AzureAppConfiguration(
		tenantID,
		clientID,
		clientSecret,
		connection,
		"",
		false,
		common.MillisecondToDuration(5000),
	)
	require.NoError(t, err)

	keyvalues := make(map[string]string)
	keyvalues["cfgkey"] = "cfgvalue!!!"
	keyvalues["cfgsecret"] = "SecretValue!!!"

	for key, value := range keyvalues {
		require.Equal(t, value, flags[key])
	}

	require.True(t, flags["cfg"] != "")
	require.False(t, common.IsValidFilename(flags["cfg"]))

	flags, err = AzureAppConfiguration(
		"23feb136-a94f-44bb-b6ff-e9d9e598f33b",
		"0385068d-4874-4a8c-a6f3-c87d0acd73b7",
		clientSecret,
		connection,
		"cfg",
		false,
		common.MillisecondToDuration(5000),
	)
	require.NoError(t, err)

	require.Equal(t, 1, len(flags))
	require.True(t, flags[common.FlagNameCfgExternal] != "")
	require.False(t, common.IsValidFilename(flags[common.FlagNameCfgExternal]))
}
