// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package cmd

import (
	//"fmt"
	"io/ioutil"
	//"net/http"
	"os"
	"testing"

	"github.com/Azure/aks-engine/pkg/api"
	"github.com/Azure/aks-engine/pkg/armhelpers"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ini "gopkg.in/ini.v1"
	//"gopkg.in/jarcoal/httpmock.v1"
)

//mockAuthProvider implements AuthProvider and allows in particular to stub out getClient()
type mockAuthProvider struct {
	getClientMock armhelpers.AKSEngineClient
	*authArgs
}

func (provider *mockAuthProvider) getClient() (armhelpers.AKSEngineClient, error) {
	if provider.getClientMock == nil {
		return &armhelpers.MockAKSEngineClient{}, nil
	}
	return provider.getClientMock, nil

}
func (provider *mockAuthProvider) getAuthArgs() *authArgs {
	return provider.authArgs
}

func TestNewRootCmd(t *testing.T) {
	command := NewRootCmd()
	if command.Use != rootName || command.Short != rootShortDescription || command.Long != rootLongDescription {
		t.Fatalf("root command should have use %s equal %s, short %s equal %s and long %s equal to %s", command.Use, rootName, command.Short, rootShortDescription, command.Long, rootLongDescription)
	}
	expectedCommands := []*cobra.Command{getCompletionCmd(command), newDeployCmd(), newGenerateCmd(), newGetVersionsCmd(), newOrchestratorsCmd(), newRotateCertsCmd(), newScaleCmd(), newUpgradeCmd(), newVersionCmd()}
	rc := command.Commands()
	for i, c := range expectedCommands {
		if rc[i].Use != c.Use {
			t.Fatalf("root command should have command %s", c.Use)
		}
	}

	command.SetArgs([]string{"--debug"})
	err := command.Execute()
	if err != nil {
		t.Fatal(err)
	}
}

func TestShowDefaultModelArg(t *testing.T) {
	command := NewRootCmd()
	command.SetArgs([]string{"--show-default-model"})
	err := command.Execute()
	if err != nil {
		t.Fatal(err)
	}
	// TODO: examine command output
}

func TestDebugArg(t *testing.T) {
	command := NewRootCmd()
	command.SetArgs([]string{"--show-default-model"})
	err := command.Execute()
	if err != nil {
		t.Fatal(err)
	}
	// TODO: examine command output
}

func TestCompletionCommand(t *testing.T) {
	command := getCompletionCmd(NewRootCmd())
	command.SetArgs([]string{})
	err := command.Execute()
	if err != nil {
		t.Fatal(err)
	}
	// TODO: examine command output
}

func TestGetSelectedCloudFromAzConfig(t *testing.T) {
	for _, test := range []struct {
		desc   string
		data   []byte
		expect string
	}{
		{"nil file", nil, "AzureCloud"},
		{"empty file", []byte{}, "AzureCloud"},
		{"no cloud section", []byte(`
		[key]
		foo = bar
		`), "AzureCloud"},
		{"cloud section empty", []byte(`
		[cloud]
		[foo]
		foo = bar
		`), "AzureCloud"},
		{"AzureCloud selected", []byte(`
		[cloud]
		name = AzureCloud
		`), "AzureCloud"},
		{"custom cloud", []byte(`
		[cloud]
		name = myCloud
		`), "myCloud"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			f, err := ini.Load(test.data)
			if err != nil {
				t.Fatal(err)
			}

			cloud := getSelectedCloudFromAzConfig(f)
			if cloud != test.expect {
				t.Fatalf("exepcted %q, got %q", test.expect, cloud)
			}
		})
	}
}

func TestGetCloudSubFromAzConfig(t *testing.T) {
	goodUUID, err := uuid.FromString("ccabad21-ea42-4ea1-affc-17ae73f9df66")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		desc   string
		data   []byte
		expect uuid.UUID
		err    bool
	}{
		{"empty file", []byte{}, uuid.UUID{}, true},
		{"no entry for cloud", []byte(`
		[SomeCloud]
		subscription = 00000000-0000-0000-0000-000000000000
		`), uuid.UUID{}, true},
		{"invalid UUID", []byte(`
		[AzureCloud]
		subscription = not-a-good-value
		`), uuid.UUID{}, true},
		{"real UUID", []byte(`
		[AzureCloud]
		subscription = ` + goodUUID.String() + `
		`), goodUUID, false},
	} {
		t.Run(test.desc, func(t *testing.T) {
			f, err := ini.Load(test.data)
			if err != nil {
				t.Fatal(err)
			}

			uuid, err := getCloudSubFromAzConfig("AzureCloud", f)
			if test.err != (err != nil) {
				t.Fatalf("expected err=%v, got: %v", test.err, err)
			}
			if test.err {
				return
			}
			if uuid.String() != test.expect.String() {
				t.Fatalf("expected %s, got %s", test.expect, uuid)
			}
		})
	}
}

func TestWriteCustomCloudProfile(t *testing.T) {
	err := prepareCustomCloudProfile()
	if err != nil {
		t.Fatalf("%v", err)
	}

	environmentFilePath := os.Getenv("AZURE_ENVIRONMENT_FILEPATH")
	if environmentFilePath == "" {
		t.Fatal("failed to write custom cloud profile: err - AZURE_ENVIRONMENT_FILEPATH is empty")
	}

	if _, err = os.Stat(environmentFilePath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		t.Fatalf("failed to write custom cloud profile: file %s does not exist", environmentFilePath)
	}

	azurestackenvironment, err := ioutil.ReadFile(environmentFilePath)
	if err != nil {
		t.Fatalf("failed to write custom cloud profile: can not read file %s ", environmentFilePath)
	}
	azurestackenvironmentStr := string(azurestackenvironment)
	expectedResult := `{"name":"azurestackcloud","managementPortalURL":"https://management.local.azurestack.external/","publishSettingsURL":"https://management.local.azurestack.external/publishsettings/index","serviceManagementEndpoint":"https://management.azurestackci15.onmicrosoft.com/36f71706-54df-4305-9847-5b038a4cf189","resourceManagerEndpoint":"https://management.local.azurestack.external/","activeDirectoryEndpoint":"https://login.windows.net/","galleryEndpoint":"https://portal.local.azurestack.external=30015/","keyVaultEndpoint":"https://vault.azurestack.external/","graphEndpoint":"https://graph.windows.net/","serviceBusEndpoint":"https://servicebus.azurestack.external/","batchManagementEndpoint":"https://batch.azurestack.external/","storageEndpointSuffix":"core.azurestack.external","sqlDatabaseDNSSuffix":"database.azurestack.external","trafficManagerDNSSuffix":"trafficmanager.cn","keyVaultDNSSuffix":"vault.azurestack.external","serviceBusEndpointSuffix":"servicebus.azurestack.external","serviceManagementVMDNSSuffix":"chinacloudapp.cn","resourceManagerVMDNSSuffix":"cloudapp.azurestack.external","containerRegistryDNSSuffix":"azurecr.io","cosmosDBDNSSuffix":"","tokenAudience":"https://management.azurestack.external/","resourceIdentifiers":{"graph":"","keyVault":"","datalake":"","batch":"","operationalInsights":""}}`
	if azurestackenvironmentStr != expectedResult {
		t.Fatalf("failed to write custom cloud profile: expected %s , got %s ", expectedResult, azurestackenvironmentStr)
	}
}

// TODO fix this
/*func TestGetAzureStackClientWithClientSecret(t *testing.T) {
	err := prepareCustomCloudProfile()
	if err != nil {
		t.Fatalf("%v", err)
	}

	subscriptionID, _ := uuid.FromString("cc6b141e-6afc-4786-9bf6-e3b9a5601460")

	for _, test := range []struct {
		desc     string
		authArgs authArgs
	}{
		{
			"identity-system azure_ad should produce valid client",
			authArgs{
				AuthMethod:          "client_secret",
				IdentitySystem:      "azure_ad",
				SubscriptionID:      subscriptionID,
				RawAzureEnvironment: "AZURESTACKCLOUD",
				ClientID:            subscriptionID,
				ClientSecret:        "secret",
			},
		},
		{
			"identity-system adfs should produce valid client",
			authArgs{
				AuthMethod:          "client_secret",
				IdentitySystem:      "adfs",
				SubscriptionID:      subscriptionID,
				RawAzureEnvironment: "AZURESTACKCLOUD",
				ClientID:            subscriptionID,
				ClientSecret:        "secret",
			},
		},
		{
			"invalid identity-system should throw error",
			authArgs{
				AuthMethod:          "client_secret",
				IdentitySystem:      "fake-system",
				RawAzureEnvironment: "AZURESTACKCLOUD",
			},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			registerRespondersForIdentitySystem(&test.authArgs)

			client, err := test.authArgs.getAzureStackClient()
			if isValidIdentitySystem(test.authArgs.IdentitySystem) {
				if client == nil {
					t.Fatalf("azure client was not created. error=%v", err)
				}
			} else {
				if err == nil || !strings.HasPrefix(err.Error(), "--auth-method") {
					t.Fatalf("failed to return error with invalid identity-system")
				}
			}
		})
	}
}

func isValidIdentitySystem(s string) bool {
	return s == "azure_ad" || s == "adfs"
}

func registerRespondersForIdentitySystem(authArgs *authArgs) {
	const (
		token         = "19590a3f-b1af-4e6b-8f63-f917cbf40711"
		tokenResponse = `
			{
				"token_type": "Bearer",
				"expires_in": "3600",
				"ext_expires_in": "3600",
				"expires_on": "1553888252",
				"not_before": "1553884352",
				"resource": "https://management.core.windows.net/",
				"access_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6Ik4tbEMwbi05REFMcXdodUhZbkhRNjNHZUNYYyIsImtpZCI6Ik4tbEMwbi05REFMcXdodUhZbkhRNjNHZUNYYyJ9.eyJhdWQiOiJodHRwczovL21hbmFnZW1lbnQuY29yZS53aW5kb3dzLm5ldC8iLCJpc3MiOiJodHRwczovL3N0cy53aW5kb3dzLm5ldC83MmY5ODhiZi04NmYxLTQxYWYtOTFhYi0yZDdjZDAxMWRiNDcvIiwiaWF0IjoxNTUzODg0MzUyLCJuYmYiOjE1NTM4ODQzNTIsImV4cCI6MTU1Mzg4ODI1MiwiYWlvIjoiNDJKZ1lHZzRIOWpPOGlCMDl4bU5JOTU3WDM4T0FBQT0iLCJhcHBpZCI6Ijg1MTE1Zjg0LWVmN2ItNGRkYi1iNDRkLWIzYTlkM2IxOTkwZCIsImFwcGlkYWNyIjoiMSIsImlkcCI6Imh0dHBzOi8vc3RzLndpbmRvd3MubmV0LzcyZjk4OGJmLTg2ZjEtNDFhZi05MWFiLTJkN2NkMDExZGI0Ny8iLCJvaWQiOiJmOWE4Y2JlZC1lOTdiLTQ0MGItYjYxNS1jNDIyOTFkOTU1NzMiLCJzdWIiOiJmOWE4Y2JlZC1lOTdiLTQ0MGItYjYxNS1jNDIyOTFkOTU1NzMiLCJ0aWQiOiI3MmY5ODhiZi04NmYxLTQxYWYtOTFhYi0yZDdjZDAxMWRiNDciLCJ1dGkiOiJIdDE0TXZkU2pFZVFfY29Ua1EwS0FBIiwidmVyIjoiMS4wIn0.sTVlgBfbztPEaN1mzRRz1W9nraI3r4jz7Kcg6gz7rGaMJT6x5gqifbeDJstUAj7au_EUhupDwD6JyKJgZY-0-IDCTYw_V4m0y_l4LQxO4STUVk86SiTGZH1gf-rXPebZ8phvk1Wgn9LpwC2gIhfoj1uSxu675-7HKwu1QZTT6m0yLMTY0CJPXQYvR2lFlZjZShJiJN1Z_zXye0K_ALv3PQwXao1buuj9PDV5GN3wolaN6DcB2gSuyAwDuD3U5Re4mpdksNs4m7O66AVfeGQV-R7ch8EW-NfFDHT3oRNjSP8WHoZjebFTg-wm2WCB7kInKRcugUo9cd-buVZARIRSAA"
			}`
		providerResponse = `
			{
				"value": [
					{
						"id": "1",
						"namespace": "Microsoft.Compute",
						"registrationState": "Registered"
					},
					{
						"id": "2",
						"namespace": "Microsoft.Storage",
						"registrationState": "Registered"},
					{
						"id": "3",
						"namespace": "Microsoft.Network",
						"registrationState": "Registered"
					}
				],
				"nextLink": "something"
			}`
	)

	switch authArgs.IdentitySystem {
	case "azure_ad":
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://management.local.azurestack.external/subscriptions/%s?api-version=2016-06-01", authArgs.SubscriptionID),
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(401, `{"error":{"code":"AuthenticationFailed","message":"Authentication failed. The 'Authorization' header is missing."}}`)
				resp.Header.Add("Www-Authenticate", fmt.Sprintf(`Bearer authorization_uri="https://login.windows.net/%s", error="invalid_token", error_description="The authentication failed because of missing 'Authorization' header."`, token))
				return resp, nil
			})

		httpmock.RegisterResponder("POST", fmt.Sprintf("https://login.windows.net/%s/oauth2/token?api-version=1.0", token),
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, tokenResponse)
				return resp, nil
			},
		)

		httpmock.RegisterResponder("GET", fmt.Sprintf("https://management.local.azurestack.external/subscriptions/%s/providers?%%24top=100&api-version=2018-05-01", authArgs.SubscriptionID),
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, providerResponse)
				return resp, nil
			})
	case "adfs":
		httpmock.RegisterResponder("POST", fmt.Sprintf("https://login.windows.net/adfs/oauth2/token?api-version=1.0"),
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, tokenResponse)
				return resp, nil
			},
		)

		httpmock.RegisterResponder("GET", fmt.Sprintf("https://management.local.azurestack.external/subscriptions/%s/providers?%%24top=100&api-version=2018-05-01", authArgs.SubscriptionID),
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, providerResponse)
				return resp, nil
			})
	}
}*/

func prepareCustomCloudProfile() error {
	const (
		name                         = "azurestackcloud"
		managementPortalURL          = "https://management.local.azurestack.external/"
		publishSettingsURL           = "https://management.local.azurestack.external/publishsettings/index"
		serviceManagementEndpoint    = "https://management.azurestackci15.onmicrosoft.com/36f71706-54df-4305-9847-5b038a4cf189"
		resourceManagerEndpoint      = "https://management.local.azurestack.external/"
		activeDirectoryEndpoint      = "https://login.windows.net/"
		galleryEndpoint              = "https://portal.local.azurestack.external=30015/"
		keyVaultEndpoint             = "https://vault.azurestack.external/"
		graphEndpoint                = "https://graph.windows.net/"
		serviceBusEndpoint           = "https://servicebus.azurestack.external/"
		batchManagementEndpoint      = "https://batch.azurestack.external/"
		storageEndpointSuffix        = "core.azurestack.external"
		sqlDatabaseDNSSuffix         = "database.azurestack.external"
		trafficManagerDNSSuffix      = "trafficmanager.cn"
		keyVaultDNSSuffix            = "vault.azurestack.external"
		serviceBusEndpointSuffix     = "servicebus.azurestack.external"
		serviceManagementVMDNSSuffix = "chinacloudapp.cn"
		resourceManagerVMDNSSuffix   = "cloudapp.azurestack.external"
		containerRegistryDNSSuffix   = "azurecr.io"
		tokenAudience                = "https://management.azurestack.external/"
	)
	cs := &api.ContainerService{
		Properties: &api.Properties{
			ServicePrincipalProfile: &api.ServicePrincipalProfile{
				ClientID: "barClientID",
				Secret:   "bazSecret",
			},
			MasterProfile: &api.MasterProfile{
				Count:     1,
				DNSPrefix: "blueorange",
				VMSize:    "Standard_D2_v2",
			},
			OrchestratorProfile: &api.OrchestratorProfile{
				OrchestratorType: api.Kubernetes,
			},
			LinuxProfile: &api.LinuxProfile{},
			CustomCloudProfile: &api.CustomCloudProfile{
				IdentitySystem:       api.AzureADIdentitySystem,
				AuthenticationMethod: api.ClientSecretAuthMethod,
				Environment: &azure.Environment{
					Name:                         name,
					ManagementPortalURL:          managementPortalURL,
					PublishSettingsURL:           publishSettingsURL,
					ServiceManagementEndpoint:    serviceManagementEndpoint,
					ResourceManagerEndpoint:      resourceManagerEndpoint,
					ActiveDirectoryEndpoint:      activeDirectoryEndpoint,
					GalleryEndpoint:              galleryEndpoint,
					KeyVaultEndpoint:             keyVaultEndpoint,
					GraphEndpoint:                graphEndpoint,
					ServiceBusEndpoint:           serviceBusEndpoint,
					BatchManagementEndpoint:      batchManagementEndpoint,
					StorageEndpointSuffix:        storageEndpointSuffix,
					SQLDatabaseDNSSuffix:         sqlDatabaseDNSSuffix,
					TrafficManagerDNSSuffix:      trafficManagerDNSSuffix,
					KeyVaultDNSSuffix:            keyVaultDNSSuffix,
					ServiceBusEndpointSuffix:     serviceBusEndpointSuffix,
					ServiceManagementVMDNSSuffix: serviceManagementVMDNSSuffix,
					ResourceManagerVMDNSSuffix:   resourceManagerVMDNSSuffix,
					ContainerRegistryDNSSuffix:   containerRegistryDNSSuffix,
					TokenAudience:                tokenAudience,
				},
			},
			AgentPoolProfiles: []*api.AgentPoolProfile{
				{
					Name:   "agentpool1",
					VMSize: "Standard_D2_v2",
					Count:  2,
				},
			},
		},
	}

	cs.SetPropertiesDefaults(false, false)

	if err := writeCustomCloudProfile(cs); err != nil {
		return errors.Wrap(err, "failed to write custom cloud profile")
	}

	return nil
}
