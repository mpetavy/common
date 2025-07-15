package azure

import (
	"fmt"
	"github.com/mpetavy/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAzureAppConfiguration(t *testing.T) {
	// CZ MED150 - Connected-Customer-Care
	tenantID, err := common.Secret("secret:+CqZUQTRaIiEz7gvBn7bgUtQGMiYfmqu1mldONDQfXYLSNsB2pUb+BdigRbpM04cqDm64Q==")
	require.NoError(t, err)
	// dataingress-app client ID
	clientID, err := common.Secret("secret:zou/xRhIxlDF1/PeCBc+9uis4w/uI/6VUMUtdO3WwaK7ZP3BjeIMISlzEHylKwHndhaAXA==")
	require.NoError(t, err)
	// dataingress-app client secret
	clientSecret, err := common.Secret("secret:y6V7pPXENeG8v5U6iA5qJQ05ZMKvI4ucotvaldpbtQ9Fcz0p0prnXpYDJG4k/god1vI7n6MM05U=")
	require.NoError(t, err)
	// dataingress-app-configuration connection
	connection, err := common.Secret("secret:MC5xBA/u9NFkkzZXLrfPVq57RfBdWJgI+hElRTNWFCGqb4RdfsthDIgAsVMAj6a/wwzFcH2GJm5U/2COKnLJjbb8L7UYJmUgzKWhLJCN68PZ6RvNrzB40L5pnkUuP2165Q4xyhEIFZMrYqzZBbFmMA2o7260sfOI6GkhuS6veAj0U8ECyB5tEPS06D09iT75t8bvvzpnU9C7SU92PRAbru/eARFYl2ijGWt4HBYq")
	require.NoError(t, err)

	azureAppCfg, err := NewAzureAppCfg(
		tenantID,
		clientID,
		clientSecret,
		connection,
	)
	require.NoError(t, err)

	flags, err := azureAppCfg.GetFlags(
		"",
		false,
		*FlagAzureTimeout,
	)
	require.NoError(t, err)

	for key, value := range flags {
		fmt.Printf("key: %s, value: %s\n", key, value)
	}

	fmt.Printf("%s\n", *common.FlagCfgExternal)

	//ctx, cancel := context.WithTimeout(context.Background(), common.MillisecondToDuration(*FlagAzureTimeout))
	//defer func() {
	//	cancel()
	//}()
	//
	//for i := range 3 {
	//	value := fmt.Sprintf("test%d", i)
	//
	//	err = azureAppCfg.SetValue(ctx, "ccc-dataingress-apikey", value)
	//	require.NoError(t, err)
	//
	//	setValue, err := azureAppCfg.GetValue(ctx, "ccc-dataingress-apikey")
	//	require.NoError(t, err)
	//
	//	require.Equal(t, setValue, value)
	//}
}
