package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/mpetavy/common"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"testing"
)

func TestAzureAppConfiguration(t *testing.T) {
	flags, err := AzureAppConfiguration(
		"23feb136-a94f-44bb-b6ff-e9d9e598f33b",
		"0385068d-4874-4a8c-a6f3-c87d0acd73b7",
		common.Secret("secret:+j3WxWEJuWvrwNoGiMiWZEHoNJH6pMXzlVbotQx6XST++EltFWn8+A+5eHT0B8N5AegBVnr2gP4="),
		common.Secret("secret:FXRnwuwS1v+ign7OTOH8usnYv5In+rPe8qQUlTvfsmobw2CEMTuuW6Wxk4aPgQiKRRXsRH5vBAy7AA8Z5rxTBeEIb0SzC4F/NjG3B3yKCZjWwUIzMluak/eWFMigxn3iOdMYfV5IF5ZVoxg3s1SbM289K6rPLix9OpU0fqCYvHZU0wcLkLCSxUaQP0dUC1ptvLdWm8zSPR/oQHJO+eQ="),
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
}

func TestChatSample(t *testing.T) {
	t.Skip("just a test sample")

	// Set up Azure credentials using the Service Principal
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")

	tenantID = *FlagAzureTenantID
	clientID = *FlagAzureClientID
	clientSecret = *FlagAzureClientSecret

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
