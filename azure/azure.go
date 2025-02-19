package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/mpetavy/common"
	"net/url"
	"os"
	"strings"
)

// https://geeksarray.com/blog/get-azure-subscription-tenant-client-id-client-secret
// https://github.com/Azure/open-service-broker-azure/issues/540

const (
	FlagNameAzureTenantID     = "azure.tenant_id"
	FlagNameAzureClientID     = "azure.client_id"
	FlagNameAzureClientSecret = "azure.client_secret"
	FlagNameAzureTimeout      = "azure.timeout"
	FlagNameAzureCfgConn      = "azure.cfg.conn"
	FlagNameAzureCfgKey       = "azure.cfg.key"
)

var (
	FlagAzureTenantID     = common.SystemFlagString(FlagNameAzureTenantID, os.Getenv("AZURE_TENANT_ID"), "Azure configuration tenant ID. Omit to use ENV parameter AZURE_TENANT_ID.")
	FlagAzureClientID     = common.SystemFlagString(FlagNameAzureClientID, os.Getenv("AZURE_CLIENT_ID"), "Azure configuration client ID. Omit to use ENV parameter AZURE_CLIENT_ID")
	FlagAzureClientSecret = common.SystemFlagString(FlagNameAzureClientSecret, os.Getenv("AZURE_CLIENT_SECRET"), "Azure configuration client secret. Omit to use ENV parameter AZURE_CLIENT_SECRET")
	FlagAzureTimeout      = common.SystemFlagInt(FlagNameAzureTimeout, 10000, "Azure timeout")
	FlagAzureCfgConn      = common.SystemFlagString(FlagNameAzureCfgConn, "", "Azure configuration connection")
	FlagAzureCfgKey       = common.SystemFlagString(FlagNameAzureCfgKey, "", "Azure configuration key name")

	azureAppCfg *AzureAppCfg
)

func init() {
	common.Events.AddListener(&common.EventExternalFlags{}, func(event common.Event) {
		ev := event.(*common.EventExternalFlags)

		if *FlagAzureCfgConn == "" {
			return
		}

		var err error

		azureAppCfg, err = NewAzureAppCfg(*FlagAzureTenantID, *FlagAzureClientID, *FlagAzureClientSecret, *FlagAzureCfgConn)
		if common.Error(err) {
			ev.Err = err

			return
		}

		flags, err := azureAppCfg.GetFlags(*FlagAzureCfgKey, true, *FlagAzureTimeout)
		if common.Error(err) {
			ev.Err = err

			return
		}

		ev.Flags = flags
	})
}

type AzureAppCfg struct {
	credentialClient azcore.TokenCredential
	configClient     *azappconfig.Client
}

func NewAzureAppCfg(tenantID string, clientID string, clientSecret string, cfgConn string) (*AzureAppCfg, error) {
	common.DebugFunc()

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

	if strings.HasPrefix(cfgConn, "Endpoint=") {
		configClient, err = azappconfig.NewClientFromConnectionString(cfgConn, nil)
		if common.Error(err) {
			return nil, err
		}
	} else {
		configClient, err = azappconfig.NewClient(cfgConn, credentialClient, nil)
		if common.Error(err) {
			return nil, err
		}
	}

	return &AzureAppCfg{
		credentialClient: credentialClient,
		configClient:     configClient,
	}, nil
}

func (azureAppCfg *AzureAppCfg) GetFlags(cfgKey string, onlyFlags bool, timeout int) (map[string]string, error) {
	common.DebugFunc()

	flags := make(map[string]string)

	ctx, cancel := context.WithTimeout(context.Background(), common.MillisecondToDuration(timeout))
	defer func() {
		cancel()
	}()

	pager := azureAppCfg.configClient.NewListSettingsPager(azappconfig.SettingSelector{}, nil)

loop:
	for {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "no more pages") {
				break
			}

			return nil, err
		}

		for _, setting := range page.Settings {
			if !onlyFlags || *setting.Key == cfgKey || common.IsValidFlagDefinition(*setting.Key, *setting.Value, true) {
				if cfgKey != "" && cfgKey != *setting.Key {
					continue
				}

				value, err := azureAppCfg.GetValue(ctx, *setting.Key)
				if common.Error(err) {
					return nil, err
				}

				if cfgKey != "" {
					flags[common.FlagNameCfgExternal] = value

					break loop
				}

				flags[*setting.Key] = value
			}
		}
	}

	common.DebugFunc("Found %d settings", len(flags))

	return flags, nil
}

func (azureAppCfg *AzureAppCfg) GetValue(ctx context.Context, key string) (string, error) {
	common.DebugFunc(key)

	resp, err := azureAppCfg.configClient.GetSetting(ctx, key, nil)
	if common.Error(err) {
		return "", err
	}

	if !strings.HasPrefix(*resp.Value, "{\"uri\":\"") {
		return *resp.Value, nil
	}

	m := make(map[string]interface{})

	err = json.Unmarshal([]byte(*resp.Value), &m)
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

	secretClient, err := azsecrets.NewClient(secretUrl.String(), azureAppCfg.credentialClient, nil)
	if common.Error(err) {
		return "", err
	}

	secretResp, err := secretClient.GetSecret(ctx, key, "", nil)
	if common.Error(err) {
		return "", err
	}

	return *secretResp.Value, nil
}

func (azureAppCfg *AzureAppCfg) SetValue(ctx context.Context, key string, value string) error {
	common.DebugFunc(key)

	resp, err := azureAppCfg.configClient.GetSetting(ctx, key, nil)
	if common.Error(err) {
		return err
	}

	if !strings.HasPrefix(*resp.Value, "{\"uri\":\"") {
		err := azureAppCfg.SetValue(ctx, key, value)
		if common.Error(err) {
			return err
		}

		return nil
	}

	m := make(map[string]interface{})

	err = json.Unmarshal([]byte(*resp.Value), &m)
	if common.Error(err) {
		return err
	}

	uri := m["uri"].(string)
	key = uri[strings.LastIndex(uri, "/")+1:]

	secretUrl, err := url.Parse(uri)
	if common.Error(err) {
		return err
	}

	// the URL in the app configuration value must consist only with Scheme and Host
	secretUrl, err = url.Parse(fmt.Sprintf("%s://%s", secretUrl.Scheme, secretUrl.Host))
	if common.Error(err) {
		return err
	}

	secretClient, err := azsecrets.NewClient(secretUrl.String(), azureAppCfg.credentialClient, nil)
	if common.Error(err) {
		return err
	}

	_, err = secretClient.SetSecret(ctx, key, azsecrets.SetSecretParameters{Value: &value}, &azsecrets.SetSecretOptions{})
	if common.Error(err) {
		return err
	}

	return nil
}
