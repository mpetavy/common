package azure

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/mpetavy/common"
	"net/url"
	"strings"
)

// https://geeksarray.com/blog/get-azure-subscription-tenant-client-id-client-secret
// https://github.com/Azure/open-service-broker-azure/issues/540

const (
	FlagNameAzureCfg = "azure.cfg"
)

var (
	FlagAzureCfgEndpoint = common.SystemFlagString(FlagNameAzureCfg, "", "Azure configuration endpoint")
)

func init() {
	common.Events.AddListener(&common.EventFlagsExternal{}, func(event common.Event) {
		if *FlagAzureCfgEndpoint == "" {
			return
		}

		flags, err := AzureAppConfiguration(true)
		common.Panic(err)

		eventConfiguraion := event.(*common.EventFlagsExternal)
		eventConfiguraion.Flags = flags
	})
}

func getValue(ctx context.Context, credentialClient azcore.TokenCredential, key string, value string) (string, error) {
	common.DebugFunc(key)

	if !strings.HasPrefix(value, "{\"uri\":\"") {
		return value, nil
	}

	m := make(map[string]interface{})

	err := json.Unmarshal([]byte(value), &m)
	if common.Error(err) {
		return "", err
	}

	uri := m["uri"].(string)
	key = uri[strings.LastIndex(uri, "/")+1:]

	secretUrl, err := url.Parse(uri)
	if common.Error(err) {
		return "", err
	}

	// the URL in the app configuration value must consist only with Scheme and Host
	secretUrl, err = url.Parse(fmt.Sprintf("%s://%s", secretUrl.Scheme, secretUrl.Host))
	if common.Error(err) {
		return "", err
	}

	secretClient, err := azsecrets.NewClient(secretUrl.String(), credentialClient, nil)
	if common.Error(err) {
		return "", err
	}

	secretResp, err := secretClient.GetSecret(ctx, key, "", nil)
	if common.Error(err) {
		return "", err
	}

	return *secretResp.Value, nil
}

func AzureAppConfiguration(onlyFlags bool) (map[string]string, error) {
	common.DebugFunc()

	ctx, cancel := context.WithTimeout(context.Background(), common.MillisecondToDuration(*FlagAzureTimeout))
	defer cancel()

	flags := make(map[string]string)

	var err error
	var credentialClient azcore.TokenCredential

	if *FlagAzureTenantID != "" {
		credentialClient, err = azidentity.NewClientSecretCredential(*FlagAzureTenantID, *FlagAzureClientID, *FlagAzureClientSecret, nil)
		if common.Error(err) {
			return nil, err
		}
	} else {
		credentialClient, err = azidentity.NewDefaultAzureCredential(nil)
		if common.Error(err) {
			return nil, err
		}
	}

	var configClient *azappconfig.Client

	if strings.HasPrefix(*FlagAzureCfgEndpoint, "Endpoint=") {
		configClient, err = azappconfig.NewClientFromConnectionString(*FlagAzureCfgEndpoint, nil)
		if common.Error(err) {
			return nil, err
		}
	} else {
		configClient, err = azappconfig.NewClient(*FlagAzureCfgEndpoint, credentialClient, nil)
		if common.Error(err) {
			return nil, err
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), common.MillisecondToDuration(*FlagAzureTimeout))
	defer func() {
		cancel()
	}()

	pager := configClient.NewListSettingsPager(azappconfig.SettingSelector{}, nil)
	for {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "no more pages") {
				break
			}

			return nil, err
		}

		for _, setting := range page.Settings {
			if !onlyFlags || (setting.Value != nil && *setting.Value != "" && flag.Lookup(*setting.Key) != nil && !common.IsCmdlineOnlyFlag(*setting.Key)) {
				value, err := getValue(ctx, credentialClient, *setting.Key, *setting.Value)
				if common.Error(err) {
					return nil, err
				}

				flags[*setting.Key] = value
			}
		}
	}

	return flags, nil
}
