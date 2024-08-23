package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/tidwall/gjson"
)

type AzureService struct {
	Id         string                 `json:"id"`
	Name       string                 `json:"name"`
	Location   string                 `json:"location"`
	Properties AzureServiceProperties `json:"properties"`
}

type AzureServiceProperties struct {
	DeveloperPortalUrl string `json:"developerPortalUrl"`
	GatewayUrl         string `json:"gatewayUrl"`
	GatewayRegionalUrl string `json:"gatewayRegionalUrl"`
	PortalUrl          string `json:"portalUrl"`
	PublisherEmail     string `json:"publisherEmail"`
	PublisherName      string `json:"publisherName"`
}

type AzureApis struct {
	Value []AzureApi `json:"value"`
}

type AzureApi struct {
	Id         string             `json:"id"`
	Type_      string             `json:"type"`
	Name       string             `json:"name"`
	Properties AzureApiProperties `json:"properties"`
}

type AzureApiProperties struct {
	DisplayName                   string                                `json:"displayName"`
	ApiRevision                   string                                `json:"apiRevision"`
	Description                   string                                `json:"description"`
	SubscriptionRequired          string                                `json:"subscriptionRequired"`
	ServiceUrl                    string                                `json:"serviceUrl"`
	BackendId                     string                                `json:"backendId"`
	Path                          string                                `json:"path"`
	Protocols                     []string                              `json:"protocols"`
	AuthenticationSettings        AzureApiAuthenticationSettings        `json:"authenticationSettings"`
	SubscriptionKeyParameterNames AzureApiSubscriptionKeyParameterNames `json:"subscriptionKeyParameterNames"`
	IsCurrent                     bool                                  `json:"isCurrent"`
	ApiRevisionDescription        string                                `json:"apiRevisionDescription"`
	ApiVersion                    string                                `json:"apiVersion"`
}

type AzureApiAuthenticationSettings struct {
	OAuth2                       string   `json:"oAuth2"`
	OpenId                       string   `json:"openId"`
	OAuth2AuthenticationSettings []string `json:"oAuth2AuthenticationSettings"`
	OpenIdAuthenticationSettings []string `json:"openIdAuthenticationSettings"`
}

type AzureApiSubscriptionKeyParameterNames struct {
	Header string `json:"header"`
	Query  string `json:"query"`
}

type AzureApiSchema struct {
	Id         string                   `json:"id"`
	Type       string                   `json:"type"`
	Name       string                   `json:"name"`
	Properties AzureApiSchemaProperties `json:"properties"`
}

type AzureApiSchemaProperties struct {
	Description string `json:"description"`
	SchemaType  string `json:"schemaType"`
	Document    string `json:"document"`
}

type AzureTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	ExpiresOn    string `json:"expires_on"`
	ExtExpiresIn string `json:"ext_expires_in"`
	NotBefore    string `json:"not_before"`
	Resource     string `json:"resource"`
	TokenType    string `json:"token_type"`
}

type ApigeeFlags struct {
	Project     string `name:"project" description:"The Google Cloud project that Apigee is running in."`
	Region      string `name:"region" description:"The Google Cloud region for a command."`
	Token       string `name:"token" description:"The Google access token to call Apigee with."`
	ApiName     string `name:"api" description:"A specific Apigee API."`
	Environment string `name:"environment" description:"A specific Apigee environment."`
}

type AzureFlags struct {
	Subscription  string `name:"subscription" description:"The Azure subscription ID."`
	ResourceGroup string `name:"resourcegroup" description:"The Azure resource group."`
	ServiceName   string `name:"name" description:"The Azure API Management service name."`
	Token         string `name:"token" description:"The Azure access token to call Azure with."`
	ApiName       string `name:"api" description:"A specific Azure API Management API."`
}

func azureServiceExport(flags *AzureFlags) error {
	var baseDir = "data/src/main/azure"
	var token string = flags.Token
	if flags.Subscription == "" {
		fmt.Println("No subscription given, cannot export Azure APIs.")
		return nil
	} else if flags.ResourceGroup == "" {
		fmt.Println("No resource group given, cannot export Azure APIs.")
		return nil
	} else if flags.ServiceName == "" {
		fmt.Println("No service name given, cannot export Azure APIs.")
		return nil
	}

	if token == "" {
		// fetch an Azure token using a client id and secret
		var env_token string = os.Getenv("AZURE_TOKEN")
		if env_token != "" {
			token = env_token
		} else {
			var client_id string = os.Getenv("AZURE_CLIENT_ID")
			var client_secret string = os.Getenv("AZURE_CLIENT_SECRET")
			var tenant_id string = os.Getenv("AZURE_TENANT_ID")

			if client_id == "" || client_secret == "" || tenant_id == "" {
				fmt.Println("No token sent and no client environment variables set, cannot export Azure APIs.")
				return nil
			}

			token = getAzureToken(client_id, client_secret, tenant_id)
		}

		if token == "" {
			fmt.Println("Could not get valid Azure token, cannot export Azure APIs.")
			return nil
		}
	}

	fmt.Println("Exporting Azure service " + flags.ServiceName + "...")
	service := getAzureService(flags.Subscription, flags.ResourceGroup, flags.ServiceName, token)
	if service != "" {
		os.MkdirAll(baseDir, 0755)
		bytes := []byte(service)
		//bytes, _ := json.MarshalIndent(service, "", " ")
		var result map[string]any
		json.Unmarshal(bytes, &result)
		bytes2, _ := json.MarshalIndent(result, "", " ")
		os.WriteFile(baseDir+"/"+flags.ServiceName+".json", bytes2, 0644)
	}

	return nil
}

func azureExport(flags *AzureFlags) error {
	var baseDir = "data/src/main/azure/apiproxies"
	var token string = flags.Token
	if flags.Subscription == "" {
		fmt.Println("No subscription given, cannot export Azure APIs.")
		return nil
	} else if flags.ResourceGroup == "" {
		fmt.Println("No resource group given, cannot export Azure APIs.")
		return nil
	} else if flags.ServiceName == "" {
		fmt.Println("No service name given, cannot export Azure APIs.")
		return nil
	}

	if token == "" {
		// fetch an Azure token using a client id and secret
		var env_token string = os.Getenv("AZURE_TOKEN")
		if env_token != "" {
			token = env_token
		} else {
			var client_id string = os.Getenv("AZURE_CLIENT_ID")
			var client_secret string = os.Getenv("AZURE_CLIENT_SECRET")
			var tenant_id string = os.Getenv("AZURE_TENANT_ID")

			if client_id == "" || client_secret == "" || tenant_id == "" {
				fmt.Println("No token sent and no client environment variables set, cannot export Azure APIs.")
				return nil
			}

			token = getAzureToken(client_id, client_secret, tenant_id)
		}

		if token == "" {
			fmt.Println("Could not get valid Azure token, cannot export Azure APIs.")
			return nil
		}
	}

	fmt.Println("Exporting Azure APIs for service " + flags.ServiceName + "...")
	apis := getAzureApis(flags.Subscription, flags.ResourceGroup, flags.ServiceName, token)
	if len(apis.Value) > 0 {
		for _, api := range apis.Value {
			if (flags.ApiName == "" || flags.ApiName == api.Name) && !strings.Contains(api.Name, ";rev=") {
				fmt.Println("Exporting " + api.Name + "...")
				bytes, _ := json.MarshalIndent(api, "", " ")
				os.RemoveAll(baseDir + "/" + api.Name)
				os.MkdirAll(baseDir+"/"+api.Name, 0755)
				os.WriteFile(baseDir+"/"+api.Name+"/"+api.Name+".json", bytes, 0644)

				schema := getAzureApiSchema(flags.Subscription, flags.ResourceGroup, flags.ServiceName, api.Name, token)

				if schema.Id != "" {
					bytes, _ := json.MarshalIndent(schema, "", " ")
					os.WriteFile(baseDir+"/"+api.Name+"/schema-definition.json", bytes, 0644)

					doc_bytes := []byte(schema.Properties.Document)
					os.WriteFile(baseDir+"/"+api.Name+"/schema."+schema.Properties.SchemaType, doc_bytes, 0644)
				}
			}
		}
	}

	return nil
}

func getAzureToken(clientId string, clientSecret string, tenantId string) string {
	var result string = ""
	var body string = "grant_type=client_credentials&client_id=" + clientId + "&client_secret=" + clientSecret + "&resource=https%3A%2F%2Fmanagement.azure.com%2F"
	bodyBuffer := bytes.NewBufferString(body)
	req, _ := http.NewRequest(http.MethodPost, "https://login.microsoftonline.com/"+tenantId+"/oauth2/token", bodyBuffer)
	response, err := http.DefaultClient.Do(req)

	//Handle Error
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	defer response.Body.Close()
	//Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var azureToken AzureTokenResponse
	json.Unmarshal(responseBody, &azureToken)

	if azureToken.AccessToken != "" {
		result = azureToken.AccessToken
	}

	return result
}

func getAzureService(subscriptionId string, resourceGroup string, serviceName string, token string) string {
	//var service AzureService
	var service string
	req, _ := http.NewRequest(http.MethodGet, "https://management.azure.com/subscriptions/"+subscriptionId+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ApiManagement/service/"+serviceName+"?api-version=2022-08-01", nil)
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			service = string(body)
			//json.Unmarshal(body, &service)
			//fmt.Println(string(body))
		}
	}

	return service
}

func getAzureApis(subscriptionId string, resourceGroup string, serviceName string, token string) AzureApis {
	var apis AzureApis
	req, _ := http.NewRequest(http.MethodGet, "https://management.azure.com/subscriptions/"+subscriptionId+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ApiManagement/service/"+serviceName+"/apis?api-version=2022-08-01", nil)
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			json.Unmarshal(body, &apis)
			//fmt.Println(string(body))
		}
	}

	return apis
}

func getAzureApiSchema(subscriptionId string, resourceGroup string, serviceName string, apiName string, token string) AzureApiSchema {
	var schema AzureApiSchema
	req, _ := http.NewRequest(http.MethodGet, "https://management.azure.com/subscriptions/"+subscriptionId+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ApiManagement/service/"+serviceName+"/schemas/"+apiName+"?api-version=2022-08-01", nil)
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				document := gjson.Get(string(body), "properties.document").String()
				json.Unmarshal(body, &schema)
				schema.Properties.Document = document
				//fmt.Println(string(body))
			}
		}
	}

	return schema
}

func azureOfframp(flags *AzureFlags) error {

	azureBaseDir := "data/src/main/azure/apiproxies"
	baseDir := "data/src/main/general/apiproxies"

	if flags.Subscription == "" {
		fmt.Println("No subscription given, cannot offramp Azure APIs.")
		return nil
	} else if flags.ResourceGroup == "" {
		fmt.Println("No resource group given, cannot offramp Azure APIs.")
		return nil
	} else if flags.ServiceName == "" {
		fmt.Println("No service name given, cannot offramp Azure APIs.")
		return nil
	}

	entries, err := os.ReadDir(azureBaseDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Offramping Azure API Management APIs to general...")

	// load azureService info, if available
	var azureService AzureService
	azureServiceFile, err := os.Open(azureBaseDir + "/../" + flags.ServiceName + ".json")

	if err == nil {
		byteValue, _ := io.ReadAll(azureServiceFile)
		json.Unmarshal(byteValue, &azureService)
	}

	for _, e := range entries {
		if flags.ApiName == "" || flags.ApiName == e.Name() {
			fmt.Println(e.Name())
			var azureApi AzureApi
			apiFile, err := os.Open(azureBaseDir + "/" + e.Name() + "/" + e.Name() + ".json")
			if err != nil {
				log.Fatal(err)
			} else {
				byteValue, _ := io.ReadAll(apiFile)
				json.Unmarshal(byteValue, &azureApi)
			}
			defer apiFile.Close()

			if azureApi.Name != "" {
				var generalApi GeneralApi
				generalApi.Name = azureApi.Name
				generalApi.DisplayName = azureApi.Properties.DisplayName
				if azureApi.Properties.ApiVersion != "" {
					generalApi.DisplayName += " " + azureApi.Properties.ApiVersion
				}
				generalApi.Description = azureApi.Properties.Description
				generalApi.Version = azureApi.Properties.ApiVersion
				generalApi.OwnerEmail = azureService.Properties.PublisherEmail
				generalApi.OwnerName = azureService.Properties.PublisherName
				generalApi.DocumentationUrl = azureService.Properties.DeveloperPortalUrl + "/api-details#api=" + e.Name()
				generalApi.GatewayUrl = azureService.Properties.GatewayUrl + "/" + azureApi.Properties.Path
				generalApi.BasePath = azureApi.Properties.Path
				generalApi.PlatformId = "azure"
				generalApi.PlatformName = "Azure API Management"
				generalApi.PlatformResourceUri = "https://portal.azure.com/#resource/subscriptions/" + flags.Subscription + "/resourceGroups/" + flags.ResourceGroup + "/providers/Microsoft.ApiManagement/service/" + flags.ServiceName + "/overview?apiName=" + e.Name()

				bytes, _ := json.MarshalIndent(generalApi, "", " ")
				os.RemoveAll(baseDir + "/" + generalApi.Name)
				os.MkdirAll(baseDir+"/"+generalApi.Name, 0755)

				os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+".json", bytes, 0644)

				schemaFile, err := os.Open(azureBaseDir + "/" + e.Name() + "/schema.json")
				if err == nil {
					// we have an api spec, copy it over
					byteValue, _ := io.ReadAll(schemaFile)
					os.WriteFile(baseDir+"/"+generalApi.Name+"/openapi.json", byteValue, 0644)
				}
			}
		}
	}

	return nil
}
