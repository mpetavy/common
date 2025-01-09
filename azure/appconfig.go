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
	"time"
)

// https://geeksarray.com/blog/get-azure-subscription-tenant-client-id-client-secret
// https://github.com/Azure/open-service-broker-azure/issues/540

func init() {
	common.Events.AddListener(&common.EventFlagsExternal{}, func(event common.Event) {
		if *FlagAzureCfg == "" {
			return
		}

		flags, err := AzureAppConfiguration(*FlagAzureTenantID, *FlagAzureClientID, *FlagAzureClientSecret, *FlagAzureCfg, true, common.MillisecondToDuration(*FlagAzureTimeout))
		common.Panic(err)

		eventFlagsExternal := event.(*common.EventFlagsExternal)
		eventFlagsExternal.Flags = flags
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

func AzureAppConfiguration(tenantID string, clientID string, clientSecret string, connection string, onlyFlags bool, timeout time.Duration) (map[string]string, error) {
	common.DebugFunc()

	flags := make(map[string]string)

	var err error
	var credentialClient azcore.TokenCredential

	if tenantID != "" {
		if clientID == "" || clientSecret == "" {
			return nil, &common.ErrFlagNotDefined{Name: strings.Join([]string{FlagNameAzureTenantID, FlagNameAzureClientID, FlagNameAzureClientSecret}, ",")}
		}

		credentialClient, err = azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
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

	if strings.HasPrefix(connection, "Endpoint=") {
		configClient, err = azappconfig.NewClientFromConnectionString(connection, nil)
		if common.Error(err) {
			return nil, err
		}
	} else {
		configClient, err = azappconfig.NewClient(connection, credentialClient, nil)
		if common.Error(err) {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
