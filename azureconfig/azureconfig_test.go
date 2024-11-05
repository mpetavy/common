package azureconfig

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"testing"
)

func TestAzureAppConfiguration(t *testing.T) {
	flags, err := AzureAppConfiguration(false)
	require.NoError(t, err)

	keyvalues := make(map[string]string)
	keyvalues["cfgkey"] = "cfgvalue!!!"
	keyvalues["cfgsecret"] = "SecretValue!!!"

	for key, value := range keyvalues {
		require.Equal(t, value, flags[key])
	}
}

func TestChatSample(t *testing.T) {
	t.Skip("just a test sample")

	// Set up Azure credentials using the Service Principal
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")

	tenantID = *FlagCfgAzureTenantID
	clientID = *FlagCfgAzureClientID
	clientSecret = *FlagCfgAzureClientSecret

	if tenantID == "" || clientID == "" || clientSecret == "" {
		log.Fatal("Environment variables AZURE_TENANT_ID, AZURE_CLIENT_ID, and AZURE_CLIENT_SECRET must be set.")
	}

	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		log.Fatalf("Failed to create Azure credentials: %v", err)
	}

	//cred, err := azidentity.NewDefaultAzureCredential(nil)
	//if err != nil {
	//	log.Fatalf("Failed to create Azure credentials: %v", err)
	//}

	//appConfigClient, err := azappconfig.NewClientFromConnectionString("Endpoint=https://mytestconfig1.azconfig.io;Id=rqdI;Secret=3RhVhr15gIW7Bq3slqJqjM4ggKWev2FImTO3p4RGaHqt3WY1JXsXJQQJ99AJACPV0roGPnw5AAACAZAC33eq", nil)
	//if common.Error(err) {
	//	log.Fatalf("Failed to create App Configuration client: %v", err)
	//}

	//Access Azure App Configuration
	appConfigClient, err := azappconfig.NewClient("https://mytestconfig1.azconfig.io", cred, nil)
	if err != nil {
		log.Fatalf("Failed to create App Configuration client: %v", err)
	}

	// Fetch a configuration setting from App Configuration
	key := "cfgkey"
	appConfigResp, err := appConfigClient.GetSetting(context.TODO(), key, nil)
	if err != nil {
		log.Fatalf("Failed to retrieve setting from App Configuration: %v", err)
	}
	fmt.Printf("App Configuration Value for key '%s': %s\n", key, *appConfigResp.Value)

	// Access Azure Key Vault
	vaultURL := "https://mytestkeyvault4config.vault.azure.net/"
	keyVaultClient, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		log.Fatalf("Failed to create Key Vault client: %v", err)
	}

	// Fetch a secret from Key Vault
	secretName := "cfgsecret"
	secretResp, err := keyVaultClient.GetSecret(context.TODO(), secretName, "", nil)
	if err != nil {
		log.Fatalf("Failed to retrieve secret from Key Vault: %v", err)
	}
	fmt.Printf("Key Vault Secret Value for '%s': %s\n", secretName, *secretResp.Value)
}
