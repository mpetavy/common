package azure

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/appconfig/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/mpetavy/common"
	"net/url"
	"strings"
	"sync"
	"time"
)

func getFlag(ctx context.Context, configClient *azappconfig.Client, credentialClient *azidentity.DefaultAzureCredential, key string) (string, error) {
	common.DebugFunc(key)
	resp, err := configClient.GetSetting(
		ctx,
		key,
		&azappconfig.GetSettingOptions{
			//Label: to.Ptr("label"),
		})
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return "", nil
		} else {
			return "", err
		}
	}

	if !strings.HasPrefix(*resp.Value, "{\"uri\":\"") {
		return *resp.Value, nil
	}

	m := make(map[string]interface{})

	err = json.Unmarshal([]byte(*resp.Value), &m)
	if common.Error(err) {
		return "", err
	}

	secretUrl, err := url.Parse(m["uri"].(string))
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

func AppConfiguration(conn string, timeout time.Duration) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	flags := make(map[string]string)

	configClient, err := azappconfig.NewClientFromConnectionString(conn, nil)
	if common.Error(err) {
		return nil, err
	}

	credentialClient, err := azidentity.NewDefaultAzureCredential(nil)
	if common.Error(err) {
		return nil, err
	}

	flagNames := []string{}
	flag.VisitAll(func(f *flag.Flag) {
		flagNames = append(flagNames, f.Name)
	})

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}

	for _, flagName := range flagNames {
		wg.Add(1)
		go func(flagName string) {
			defer func() {
				wg.Done()
			}()

			value, _ := getFlag(ctx, configClient, credentialClient, flagName)

			mu.Lock()
			if value != "" {
				flags[flagName] = value
			}
			mu.Unlock()
		}(flagName)
	}

	wg.Wait()

	fmt.Printf("%+v\n", flags)

	return flags, nil
}
