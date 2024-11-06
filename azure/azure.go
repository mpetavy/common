package azure

import (
	"github.com/mpetavy/common"
	"os"
)

// https://geeksarray.com/blog/get-azure-subscription-tenant-client-id-client-secret
// https://github.com/Azure/open-service-broker-azure/issues/540

const (
	FlagNameAzureTenantID     = "azure.tenant_id"
	FlagNameAzureClientID     = "azure.client_id"
	FlagNameAzureClientSecret = "azure.client_secret"
	FlagNameAzureTimeout      = "azure.timeout"
)

var (
	FlagAzureTenantID     = common.SystemFlagString(FlagNameAzureTenantID, os.Getenv("AZURE_TENANT_ID"), "Azure configuration tenant ID. Omit to use ENV.")
	FlagAzureClientID     = common.SystemFlagString(FlagNameAzureClientID, os.Getenv("AZURE_CLIENT_ID"), "Azure configuration client ID. Omit to use ENV.")
	FlagAzureClientSecret = common.SystemFlagString(FlagNameAzureClientSecret, os.Getenv("AZURE_CLIENT_SECRET"), "Azure configuration client secret. Omit to use ENV.")
	FlagAzureTimeout      = common.SystemFlagInt(FlagNameAzureTimeout, 30000, "Azure configuration timeout")
)
