package main

import (
	"archive/zip"
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/leaanthony/clir"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GeneralApi struct {
	Name                string `json:"name"`
	DisplayName         string `json:"displayName"`
	Version             string `json:"version"`
	Description         string `json:"description"`
	OwnerEmail          string `json:"ownerEmail"`
	OwnerName           string `json:"ownerName"`
	DocumentationUrl    string `json:"documentationUrl"`
	GatewayUrl          string `json:"gatewayUrl"`
	PlatformId          string `json:"platformId"`
	PlatformName        string `json:"platformName"`
	PlatformResourceUri string `json:"platformResourceUri"`
}

type ApigeeProxies struct {
	Proxies []ApigeeApi `json:"proxies"`
}

type ApigeeApi struct {
	Name         string   `json:"name"`
	Revision     []string `json:"revision"`
	ApiProxyType string   `json:"apiProxyType"`
}

type ApigeeEnvironment struct {
	Proxies     []ApigeeEnvironmentProxy `json:"proxies"`
	SharedFlows []ApigeeEnvironmentProxy `json:"sharedflows"`
}

type ApigeeEnvironmentProxy struct {
	Name string `json:"name"`
}

type ApigeeDeveloper struct {
	Email     string `json:"email"`
	UserName  string `json:"userName"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type ApigeeDeveloperApp struct {
	DeveloperEmail string   `json:"developerEmail"`
	Name           string   `json:"name"`
	DisplayName    string   `json:"displayName"`
	ApiProducts    []string `json:"apiProducts"`
	ExpiryType     string   `json:"expiryType"`
}

type ApigeeProduct struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	Scopes       []string `json:"scopes"`
	Environments []string `json:"environments"`
	ApiResources []string `json:"apiResources"`
	Proxies      []string `json:"proxies"`
}

type HubApis struct {
	Apis []HubApi `json:"apis"`
}

type HubApi struct {
	Name          string              `json:"name"`
	DisplayName   string              `json:"displayName"`
	Description   string              `json:"description"`
	Documentation HubApiDocumentation `json:"documentation"`
	Owner         HubApiOwner         `json:"owner"`
	Versions      []string            `json:"versions"`
}

type HubApiDocumentation struct {
	ExternalUri string `json:"externalUri"`
}

type HubApiOwner struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

type HubApiDeployments struct {
	Deployments []HubApiDeployment `json:"deployments"`
}

type HubApiDeployment struct {
	Name           string              `json:"name"`
	DisplayName    string              `json:"displayName"`
	Description    string              `json:"description"`
	Documentation  HubApiDocumentation `json:"documentation"`
	DeploymentType HubAttribute        `json:"deploymentType"`
	ResourceUri    string              `json:"resourceUri"`
	Endpoints      []string            `json:"endpoints"`
	ApiVersions    []string            `json:"apiVersions"`
}

type HubApiVersion struct {
	Name          string              `json:"name"`
	DisplayName   string              `json:"displayName"`
	Description   string              `json:"description"`
	Documentation HubApiDocumentation `json:"documentation"`
	Deployments   []string            `json:"deployments"`
}

type HubAttribute struct {
	Attribute  string                 `json:"attribute"`
	EnumValues HubAttributeEnumValues `json:"enumValues"`
}

type HubAttributeEnumValues struct {
	Values []HubAttributeValue `json:"values"`
}

type HubAttributeValue struct {
	Id          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Immutable   bool   `json:"immutable"`
}

type HubApiVersionSpec struct {
	Name          string              `json:"name"`
	DisplayName   string              `json:"displayName"`
	SpecType      HubAttribute        `json:"specType"`
	Contents      HubContents         `json:"contents"`
	Documentation HubApiDocumentation `json:"documentation"`
}

type HubContents struct {
	MimeType string `json:"mimeType"`
	Contents string `json:"contents"`
}

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

func main() {
	// Create new cli
	cli := clir.NewCli("apimsync", "A syncing tool between API platforms", "v0.0.1")

	apigeeCommand := cli.NewSubCommand("apigee", "Functions for Apigee.")
	apigeeApisCommand := apigeeCommand.NewSubCommand("apis", "Functions for Apigee API resources.")
	apigeeApisCommand.NewSubCommandFunction("export", "Exports Apigee APIs from a given project.", apigeeExport)
	apigeeApisCommand.NewSubCommandFunction("import", "Imports APIs to an Apigee project.", apigeeImport)
	apigeeApisCommand.NewSubCommandFunction("clean", "Removes all of the Apigee APIs from a given project.", apigeeClean)
	apigeeTestCommand := apigeeCommand.NewSubCommand("test", "Local test commands.")
	apigeeTestCommand.NewSubCommandFunction("init", "Initializes local test data for an environment.", initApigeeTest)

	azureCommand := cli.NewSubCommand("azure", "Functions for Azure API Management.")
	azureCommand.NewSubCommandFunction("export", "Exports Azure API Management service info.", azureServiceExport)
	azureApisCommand := azureCommand.NewSubCommand("apis", "Functions for Azure API Management API resources.")
	azureApisCommand.NewSubCommandFunction("export", "Exports Azure API Management APIs.", azureExport)
	azureApisCommand.NewSubCommandFunction("offramp", "Migrates Azure API Management APIs out to general.", azureOfframp)

	apiHubCommand := cli.NewSubCommand("apihub", "Functions for Apigee API Hub.")
	apiHubApisCommand := apiHubCommand.NewSubCommand("apis", "Functions for API Hub API resources.")
	apiHubApisCommand.NewSubCommandFunction("onramp", "Onramps APIs from general to API Hub.", apiHubOnramp)
	apiHubApisCommand.NewSubCommandFunction("import", "Imports APIs to API Hub.", apiHubImport)
	apiHubApisCommand.NewSubCommandFunction("clean", "Imports APIs to API Hub.", apiHubClean)

	err := cli.Run()

	if err != nil {
		// We had an error
		log.Fatal(err)
	}
}

func apigeeExport(flags *ApigeeFlags) error {
	if flags.Project == "" {
		fmt.Println("No project given, cannot export Apigee APIs.")
		return nil
	}

	fmt.Println("Exporting Apigee APIs for project " + flags.Project + "...")
	var baseDir = "data/src/main/apigee/apiproxies"
	var environment ApigeeEnvironment
	if flags.Environment != "" {
		// Create dir if it does not exist
		os.MkdirAll("data/src/main/apigee/environments/"+flags.Environment, 0755)

		// Open deployments.json file
		deploymentsFile, err := os.Open("data/src/main/apigee/environments/" + flags.Environment + "/deployments.json")
		if err != nil {
			environment = ApigeeEnvironment{Proxies: []ApigeeEnvironmentProxy{}, SharedFlows: []ApigeeEnvironmentProxy{}}
		} else {
			byteValue, _ := io.ReadAll(deploymentsFile)
			json.Unmarshal(byteValue, &environment)
		}
		defer deploymentsFile.Close()
	}

	if flags.Token == "" {
		var token *oauth2.Token
		scopes := []string{
			"https://www.googleapis.com/auth/cloud-platform",
		}

		ctx := context.Background()
		credentials, err := google.FindDefaultCredentials(ctx, scopes...)

		if err == nil {
			token, err = credentials.TokenSource.Token()

			if err == nil {
				flags.Token = token.AccessToken
			}
		}
	}

	apis := getApigeeApis(flags.Project, flags.Token)
	// apisOutput, _ := json.Marshal(apis)
	// fmt.Println(string(apisOutput))

	if apis.Proxies != nil {
		os.MkdirAll(baseDir, 0755)
		for _, api := range apis.Proxies {
			if flags.ApiName == "" || flags.ApiName == api.Name {
				fmt.Println("Exporting " + api.Name + "...")
				bundle := getApigeeApiBundle(flags.Project, api.Name, api.Revision[0], flags.Token)

				if bundle != nil {
					err := os.WriteFile(baseDir+"/"+api.Name+".zip", bundle, 0644)
					if err != nil {
						panic(err)
					}

					// extract zip file
					unzipApigeeBundle(baseDir, api.Name)

					err = os.Remove(baseDir + "/" + api.Name + ".zip")
					if err != nil {
						panic(err)
					}

					// add to deployments.json if not already there
					foundProxy := false
					for _, value := range environment.Proxies {
						if value.Name == api.Name {
							foundProxy = true
						}
					}
					if !foundProxy {
						// add to deployments.json
						environment.Proxies = append(environment.Proxies, ApigeeEnvironmentProxy{Name: api.Name})
					}
				}
			}
		}

		if flags.Environment != "" {
			// write deployments.json
			bytes, _ := json.MarshalIndent(environment, "", " ")
			os.WriteFile("data/src/main/apigee/environments/"+flags.Environment+"/deployments.json", bytes, 0644)
		}
	}

	return nil
}

func apigeeImport(flags *ApigeeFlags) error {
	if flags.Project == "" {
		fmt.Println("No project given.")
		return nil
	}

	fmt.Println("Importing Apigee APIs to project " + flags.Project + "...")
	var baseDir = "data/src/main/apigee/apiproxies"
	if flags.Token == "" {
		var token *oauth2.Token
		scopes := []string{
			"https://www.googleapis.com/auth/cloud-platform",
		}

		ctx := context.Background()
		credentials, err := google.FindDefaultCredentials(ctx, scopes...)

		if err == nil {
			token, err = credentials.TokenSource.Token()

			if err == nil {
				flags.Token = token.AccessToken
			}
		}
	}

	apis, err := os.ReadDir(baseDir)
	if err == nil {
		for _, e := range apis {
			if flags.ApiName == "" || flags.ApiName == e.Name() {
				fmt.Println("Importing " + e.Name() + "...")
				os.Chdir(baseDir + "/" + e.Name())
				zipApigeeBundle(e.Name())
				err := createApigeeApi(flags.Project, flags.Token, e.Name())
				if err != nil {
					fmt.Println("Error importing Apigee API: " + err.Error())
				}
				os.Remove(e.Name() + ".zip")
				os.Chdir(baseDir + "../../../../..")
			}
		}
	}

	return nil
}

func apigeeClean(flags *ApigeeFlags) error {
	if flags.Project == "" {
		fmt.Println("No project given.")
		return nil
	}

	fmt.Println("Removing all Apigee APIs for project " + flags.Project + "...")

	if flags.Token == "" {
		var token *oauth2.Token
		scopes := []string{
			"https://www.googleapis.com/auth/cloud-platform",
		}

		ctx := context.Background()
		credentials, err := google.FindDefaultCredentials(ctx, scopes...)

		if err == nil {
			token, err = credentials.TokenSource.Token()

			if err == nil {
				flags.Token = token.AccessToken
			}
		}
	}

	apis := getApigeeApis(flags.Project, flags.Token)
	for _, api := range apis.Proxies {
		if flags.ApiName == "" || flags.ApiName == api.Name {
			fmt.Println("Deleting " + api.Name + "...")
			deleteApigeeApi(flags.Project, flags.Token, api.Name)
		}
	}

	return nil
}

func getApigeeApis(org string, token string) ApigeeProxies {
	var apis ApigeeProxies
	req, _ := http.NewRequest(http.MethodGet, "https://apigee.googleapis.com/v1/organizations/"+org+"/apis?includeRevisions=true", nil)
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

func getApigeeApiBundle(org string, api string, revision string, token string) []byte {
	var bundle []byte

	req, _ := http.NewRequest(http.MethodGet, "https://apigee.googleapis.com/v1/organizations/"+org+"/apis/"+api+"/revisions/"+revision+"?format=bundle", nil)
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		bundle, _ = io.ReadAll(resp.Body)
	}

	return bundle
}

func unzipApigeeBundle(basePath string, name string) {
	zipBaseBath := basePath + "/" + name
	os.Mkdir(basePath+"/"+name, 0755)
	archive, err := zip.OpenReader(basePath + "/" + name + ".zip")
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(zipBaseBath, f.Name)

		if !strings.HasPrefix(filePath, filepath.Clean(zipBaseBath)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
}

func zipApigeeBundle(name string) {
	file, err := os.Create(name + ".zip")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Ensure that `path` is not absolute; it should not start with "/".
		// This snippet happens to work because I don't use
		// absolute paths, but ensure your real-world code
		// transforms path into a zip-root relative path.
		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}

	err = filepath.Walk("apiproxy", walker)
	if err != nil {
		panic(err)
	}
}

func deleteApigeeApi(org string, token string, api string) {
	req, _ := http.NewRequest(http.MethodDelete, "https://apigee.googleapis.com/v1/organizations/"+org+"/apis/"+api, nil)
	req.Header.Add("Authorization", "Bearer "+token)

	_, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error deleting Apigee API: " + err.Error())
	}
}

func createApigeeApi(org string, token string, name string) error {

	fileDir, _ := os.Getwd()
	fileName := name + ".zip"
	filePath := path.Join(fileDir, fileName)

	file, _ := os.Open(filePath)
	defer file.Close()

	fmt.Println(filepath.Base(file.Name()))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.Close()

	r, _ := http.NewRequest(http.MethodPost, "https://apigee.googleapis.com/v1/organizations/"+org+"/apis?name="+name+"&action=import", body)
	r.Header.Add("Content-Type", writer.FormDataContentType())
	r.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(r)

	if resp.StatusCode != 200 {
		fmt.Println("Error creating Apigee API: " + resp.Status)
	}

	return err
}

func initApigeeTest(flags *ApigeeFlags) error {
	if flags.Project == "" {
		fmt.Println("No project given, cannot init test data.")
		return nil
	}

	if flags.Environment == "" {
		fmt.Println("No environment given, cannot init test data.")
		return nil
	}

	// create test developer
	developer := ApigeeDeveloper{Email: "test@example.com", UserName: "testUser", FirstName: "Test", LastName: "User"}
	developers := []ApigeeDeveloper{developer}

	// create test product
	product := ApigeeProduct{Name: "test_product", DisplayName: "Test Product", Scopes: []string{}, Environments: []string{}, ApiResources: []string{"/"}, Proxies: []string{}}
	products := []ApigeeProduct{product}

	// create test developerapp
	app := ApigeeDeveloperApp{DeveloperEmail: developer.Email, Name: "test_app", DisplayName: "Test App", ApiProducts: []string{"test_product"}, ExpiryType: "never"}
	apps := []ApigeeDeveloperApp{app}

	// load environment deployments.json
	var environment ApigeeEnvironment
	deploymentsFile, err := os.Open("data/src/main/apigee/environments/" + flags.Environment + "/deployments.json")
	if err != nil {
		environment = ApigeeEnvironment{Proxies: []ApigeeEnvironmentProxy{}, SharedFlows: []ApigeeEnvironmentProxy{}}
	} else {
		byteValue, _ := io.ReadAll(deploymentsFile)
		json.Unmarshal(byteValue, &environment)
	}
	defer deploymentsFile.Close()

	// create test directory
	os.MkdirAll("data/src/main/apigee/tests/"+flags.Environment, 0755)

	// write developers
	bytes, _ := json.MarshalIndent(developers, "", " ")
	os.WriteFile("data/src/main/apigee/tests/"+flags.Environment+"/developers.json", bytes, 0644)

	for _, proxy := range environment.Proxies {
		products[0].Proxies = append(products[0].Proxies, proxy.Name)
	}

	// write products
	bytes, _ = json.MarshalIndent(products, "", " ")
	os.WriteFile("data/src/main/apigee/tests/"+flags.Environment+"/products.json", bytes, 0644)

	// write apps
	bytes, _ = json.MarshalIndent(apps, "", " ")
	os.WriteFile("data/src/main/apigee/tests/"+flags.Environment+"/developerapps.json", bytes, 0644)

	return nil
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
				generalApi.Description = azureApi.Properties.Description
				generalApi.Version = azureApi.Properties.ApiRevision
				generalApi.OwnerEmail = azureService.Properties.PublisherEmail
				generalApi.OwnerName = azureService.Properties.PublisherName
				generalApi.DocumentationUrl = azureService.Properties.DeveloperPortalUrl + "/api-details#api=" + e.Name()
				generalApi.GatewayUrl = azureService.Properties.GatewayUrl + "/" + azureApi.Properties.Path
				generalApi.PlatformId = "azure"
				generalApi.PlatformName = "Azure API Management"
				generalApi.PlatformResourceUri = "https://portal.azure.com/#resource/subscriptions/" + flags.Subscription + "/resourceGroups/" + flags.ResourceGroup + "/providers/Microsoft.ApiManagement/service/" + flags.ServiceName + "/overview"

				bytes, _ := json.MarshalIndent(generalApi, "", " ")
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

func apiHubOnramp(flags *ApigeeFlags) error {
	generalBaseDir := "data/src/main/general/apiproxies"
	baseDir := "data/src/main/apihub/apiproxies"

	if flags.Project == "" {
		fmt.Println("No project given.")
		return nil
	} else if flags.Region == "" {
		fmt.Println("No region given.")
		return nil
	}

	if flags.Token == "" {
		var token *oauth2.Token
		scopes := []string{
			"https://www.googleapis.com/auth/cloud-platform",
		}

		ctx := context.Background()
		credentials, err := google.FindDefaultCredentials(ctx, scopes...)

		if err == nil {
			token, err = credentials.TokenSource.Token()

			if err == nil {
				flags.Token = token.AccessToken
			}
		}
	}

	entries, err := os.ReadDir(generalBaseDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Onramping APIs to API Hub...")

	for _, e := range entries {
		if flags.ApiName == "" || flags.ApiName == e.Name() {
			fmt.Println(e.Name())
			var generalApi GeneralApi
			apiFile, err := os.Open(generalBaseDir + "/" + e.Name() + "/" + e.Name() + ".json")
			if err != nil {
				log.Fatal(err)
			} else {
				byteValue, _ := io.ReadAll(apiFile)
				json.Unmarshal(byteValue, &generalApi)
			}
			defer apiFile.Close()

			if generalApi.Name != "" {
				os.MkdirAll(baseDir+"/"+generalApi.Name, 0755)

				var hubApi HubApi
				hubApi.DisplayName = generalApi.DisplayName
				hubApi.Description = generalApi.Description
				hubApi.Documentation.ExternalUri = generalApi.DocumentationUrl
				hubApi.Owner.DisplayName = generalApi.OwnerName
				hubApi.Owner.Email = generalApi.OwnerEmail
				bytes, _ := json.MarshalIndent(hubApi, "", " ")
				os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+".json", bytes, 0644)

				var hubApiDeployment HubApiDeployment
				hubApiDeployment.Name = "projects/" + flags.Project + "/locations/" + flags.Region + "/deployments/" + generalApi.Name + "-" + generalApi.Version
				hubApiDeployment.DisplayName = generalApi.DisplayName
				hubApiDeployment.Description = generalApi.Description
				hubApiDeployment.Documentation.ExternalUri = generalApi.DocumentationUrl
				hubApiDeployment.DeploymentType.Attribute = "projects/" + flags.Project + "/locations/" + flags.Region + "/attributes/system-deployment-type"
				platformId := "others"
				platformName := "Others"
				platformDescription := "Others"
				apiDeploymentType := HubAttributeValue{Id: platformId, DisplayName: platformName, Description: platformDescription, Immutable: true}
				hubApiDeployment.DeploymentType.EnumValues.Values = append(hubApiDeployment.DeploymentType.EnumValues.Values, apiDeploymentType)
				hubApiDeployment.ResourceUri = generalApi.PlatformResourceUri
				hubApiDeployment.Endpoints = append(hubApiDeployment.Endpoints, generalApi.GatewayUrl)
				hubApiDeployment.ApiVersions = append(hubApiDeployment.ApiVersions, generalApi.Version)
				bytes, _ = json.MarshalIndent(hubApiDeployment, "", " ")
				os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+"-deployment.json", bytes, 0644)

				var hubApiVersion HubApiVersion
				hubApiVersion.Name = "projects/" + flags.Project + "/locations/" + flags.Region + "/versions/" + generalApi.Name + "-" + generalApi.Version
				hubApiVersion.DisplayName = generalApi.DisplayName
				hubApiVersion.Description = generalApi.Description
				hubApiVersion.Documentation.ExternalUri = generalApi.DocumentationUrl
				hubApiVersion.Deployments = append(hubApiVersion.Deployments, hubApiDeployment.Name)
				bytes, _ = json.MarshalIndent(hubApiVersion, "", " ")
				os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+"-version.json", bytes, 0644)

				// load API spec, if available
				b, err := os.ReadFile(generalBaseDir + "/" + generalApi.Name + "/openapi.json")
				if err == nil {
					// we have a spec file
					var hubApiVersionSpec HubApiVersionSpec
					hubApiVersionSpec.Name = "projects/" + flags.Project + "/locations/" + flags.Region + "/versions/" + generalApi.Name + "/specs/" + generalApi.Name + "-" + generalApi.Version
					hubApiVersionSpec.DisplayName = generalApi.DisplayName
					apiSpecType := HubAttributeValue{Id: "openapi", DisplayName: "OpenAPI Spec", Description: "OpenAPI Spec", Immutable: true}
					hubApiVersionSpec.SpecType.EnumValues.Values = append(hubApiVersionSpec.SpecType.EnumValues.Values, apiSpecType)
					hubApiVersionSpec.Contents.MimeType = "application/json"
					hubApiVersionSpec.Contents.Contents = b64.StdEncoding.EncodeToString(b)
					hubApiVersionSpec.Documentation.ExternalUri = generalApi.DocumentationUrl
					bytes, _ = json.MarshalIndent(hubApiVersionSpec, "", " ")
					os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+"-version-spec.json", bytes, 0644)
				}
			}
		}
	}

	return nil
}

func apiHubImport(flags *ApigeeFlags) error {
	if flags.Project == "" {
		fmt.Println("No project given.")
		return nil
	} else if flags.Region == "" {
		fmt.Println("No region given.")
		return nil
	}

	fmt.Println("Importing APIs to API Hub in project " + flags.Project + "...")
	var baseDir = "data/src/main/apihub/apiproxies"
	if flags.Token == "" {
		var token *oauth2.Token
		scopes := []string{
			"https://www.googleapis.com/auth/cloud-platform",
		}

		ctx := context.Background()
		credentials, err := google.FindDefaultCredentials(ctx, scopes...)

		if err == nil {
			token, err = credentials.TokenSource.Token()

			if err == nil {
				flags.Token = token.AccessToken
			}
		}
	}

	apis, err := os.ReadDir(baseDir)
	if err == nil {
		for _, e := range apis {
			if flags.ApiName == "" || flags.ApiName == e.Name() {
				fmt.Println("Importing " + e.Name() + "...")
				var versionedName = e.Name()
				// Create API
				apiFile, err := os.Open(baseDir + "/" + e.Name() + "/" + e.Name() + ".json")
				if err == nil {
					r, _ := http.NewRequest(http.MethodPost, "https://apihub.googleapis.com/v1/projects/"+flags.Project+"/locations/"+flags.Region+"/apis?apiId="+e.Name(), apiFile)
					r.Header.Add("Content-Type", "application/json")
					r.Header.Add("Authorization", "Bearer "+flags.Token)
					client := &http.Client{}
					resp, _ := client.Do(r)

					if resp.StatusCode != 200 {
						fmt.Println("  >> Error creating " + e.Name() + ": " + resp.Status)
					}
				}
				defer apiFile.Close()

				// Create Deployment
				deploymentFile, deployErr := os.Open(baseDir + "/" + e.Name() + "/" + e.Name() + "-deployment.json")
				if deployErr == nil {
					var apiDeployment HubApiDeployment
					byteValue, _ := io.ReadAll(deploymentFile)
					json.Unmarshal(byteValue, &apiDeployment)
					requestBody := bytes.NewBuffer(byteValue)
					versionedName += "-" + apiDeployment.ApiVersions[0]
					deploymentUrl := "https://apihub.googleapis.com/v1/projects/" + flags.Project + "/locations/" + flags.Region + "/deployments?deploymentId=" + versionedName
					r, _ := http.NewRequest(http.MethodPost, deploymentUrl, requestBody)
					r.Header.Add("Content-Type", "application/json")
					r.Header.Add("Authorization", "Bearer "+flags.Token)
					client := &http.Client{}
					resp, _ := client.Do(r)

					if resp.StatusCode != 200 {
						fmt.Println("  >> Error deploying " + e.Name() + "-" + apiDeployment.ApiVersions[0] + ": " + resp.Status)
						defer resp.Body.Close()
						//Read the response body
						respBody, _ := io.ReadAll(resp.Body)
						sb := string(respBody)
						fmt.Println(sb)
					}
				}
				defer deploymentFile.Close()

				// Create API Version
				versionFile, err := os.Open(baseDir + "/" + e.Name() + "/" + e.Name() + "-version.json")
				if err == nil {
					var apiVersion HubApiVersion
					byteValue, _ := io.ReadAll(versionFile)
					json.Unmarshal(byteValue, &apiVersion)
					requestBody := bytes.NewBuffer(byteValue)

					versionUrl := "https://apihub.googleapis.com/v1/projects/" + flags.Project + "/locations/" + flags.Region + "/apis/" + e.Name() + "/versions?versionId=" + versionedName
					r, _ := http.NewRequest(http.MethodPost, versionUrl, requestBody)
					r.Header.Add("Content-Type", "application/json")
					r.Header.Add("Authorization", "Bearer "+flags.Token)
					client := &http.Client{}
					resp, _ := client.Do(r)

					if resp.StatusCode != 200 {
						fmt.Println("  >> Error deploying version " + versionedName + ": " + resp.Status)
						defer resp.Body.Close()
						//Read the response body
						respBody, _ := io.ReadAll(resp.Body)
						sb := string(respBody)
						fmt.Println(sb)
					}
				}
				defer versionFile.Close()

				// Create API Version Spec
				versionSpecFile, err := os.Open(baseDir + "/" + e.Name() + "/" + e.Name() + "-version-spec.json")
				if err == nil {
					var apiVersionSpec HubApiVersionSpec
					byteValue, _ := io.ReadAll(versionSpecFile)
					json.Unmarshal(byteValue, &apiVersionSpec)
					requestBody := bytes.NewBuffer(byteValue)

					versionUrl := "https://apihub.googleapis.com/v1/projects/" + flags.Project + "/locations/" + flags.Region + "/apis/" + e.Name() + "/versions/" + versionedName + "/specs?specId=" + versionedName
					r, _ := http.NewRequest(http.MethodPost, versionUrl, requestBody)
					r.Header.Add("Content-Type", "application/json")
					r.Header.Add("Authorization", "Bearer "+flags.Token)
					client := &http.Client{}
					resp, _ := client.Do(r)

					if resp.StatusCode != 200 {
						fmt.Println("  >> Error deploying version spec " + versionedName + ": " + resp.Status)
						defer resp.Body.Close()
						//Read the response body
						respBody, _ := io.ReadAll(resp.Body)
						sb := string(respBody)
						fmt.Println(sb)
					}
				}
				defer versionSpecFile.Close()
			}
		}
	}

	return nil
}

func apiHubClean(flags *ApigeeFlags) error {
	if flags.Project == "" {
		fmt.Println("No project given.")
		return nil
	} else if flags.Region == "" {
		fmt.Println("No region given.")
		return nil
	}

	fmt.Println("Removing all API Hub APIs for project " + flags.Project + "...")

	if flags.Token == "" {
		var token *oauth2.Token
		scopes := []string{
			"https://www.googleapis.com/auth/cloud-platform",
		}

		ctx := context.Background()
		credentials, err := google.FindDefaultCredentials(ctx, scopes...)

		if err == nil {
			token, err = credentials.TokenSource.Token()

			if err == nil {
				flags.Token = token.AccessToken
			}
		}
	}

	apis := getApiHubApis(flags.Project, flags.Region, flags.Token)
	for _, api := range apis.Apis {
		if flags.ApiName == "" || strings.HasSuffix(api.Name, "/"+flags.ApiName) {
			fmt.Println("Deleting " + api.Name + "...")
			deleteApiHubApi(api.Name, flags.Token)
		}
	}

	deployments := getApiHubDeployments(flags.Project, flags.Region, flags.Token)
	for _, deployment := range deployments.Deployments {
		fmt.Println("Deleting " + deployment.Name + "...")
		deleteApiHubDeployment(deployment.Name, flags.Token)
	}

	return nil
}

func getApiHubApis(project string, region string, token string) HubApis {
	var apis HubApis

	req, _ := http.NewRequest(http.MethodGet, "https://apihub.googleapis.com/v1/projects/"+project+"/locations/"+region+"/apis", nil)
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

func deleteApiHubApi(api string, token string) {
	req, _ := http.NewRequest(http.MethodDelete, "https://apihub.googleapis.com/v1/"+api+"?force=true", nil)
	req.Header.Add("Authorization", "Bearer "+token)

	_, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error deleting Apigee API: " + err.Error())
	}
}

func getApiHubDeployments(project string, region string, token string) HubApiDeployments {
	var deployments HubApiDeployments

	req, _ := http.NewRequest(http.MethodGet, "https://apihub.googleapis.com/v1/projects/"+project+"/locations/"+region+"/deployments", nil)
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			json.Unmarshal(body, &deployments)
			//fmt.Println(string(body))
		}
	}

	return deployments
}

func deleteApiHubDeployment(deployment string, token string) {
	req, _ := http.NewRequest(http.MethodDelete, "https://apihub.googleapis.com/v1/"+deployment, nil)
	req.Header.Add("Authorization", "Bearer "+token)

	_, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error deleting API Hub Deployment: " + err.Error())
	}
}
