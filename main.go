package main

import (
	"archive/zip"
	"bytes"
	"context"
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
	Token       string `name:"token" description:"The Google access token to call Apigee with."`
	ApiName     string `name:"api" description:"A specific Apigee API."`
	Environment string `name:"environment" description:"A specific Apigee environment."`
}

type AzureFlags struct {
	Subscription  string `name:"subscription" description:"The Azure subscription ID."`
	ResourceGroup string `name:"resourcegroup" description:"The Azure resource group."`
	ServiceName   string `name:"name" description:"The Azure API Management service name."`
	Token         string `name:"token" description:"The Azure access token to call Azure with."`
}

func main() {
	// Create new cli
	cli := clir.NewCli("apimsync", "A syncing tool between API platforms", "v0.0.1")

	apigeeCommand := cli.NewSubCommand("apigee", "Functions for Apigee APIs.")
	apigeeCommand.NewSubCommandFunction("export", "Exports Apigee APIs from a given project.", apigeeExport)
	apigeeCommand.NewSubCommandFunction("import", "Imports APIs to an Apigee project.", apigeeImport)
	apigeeCommand.NewSubCommandFunction("clean", "Removes all of the Apigee APIs from a given project.", apigeeClean)
	apigeeTestCommand := apigeeCommand.NewSubCommand("test", "Local test commands.")
	apigeeTestCommand.NewSubCommandFunction("init", "Initializes local test data for an environment.", initApigeeTest)

	azureCommand := cli.NewSubCommand("azure", "Functions for Azure API Management APIs.")
	azureCommand.NewSubCommandFunction("export", "Exports Apigee APIs from a given project.", azureExport)

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
	var baseDir = "data/" + flags.Project + "/src/main/apigee/apiproxies"
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
			if !strings.Contains(api.Name, ";rev=") {
				fmt.Println("Exporting " + api.Name + "...")
				bytes, _ := json.MarshalIndent(api, "", " ")
				os.MkdirAll(baseDir+"/"+api.Name, 0755)
				os.WriteFile(baseDir+"/"+api.Name+"/"+api.Name+".json", bytes, 0644)

				schema := getAzureApiSchema(flags.Subscription, flags.ResourceGroup, flags.ServiceName, api.Name, token)

				if schema.Id != "" {
					bytes, _ := json.MarshalIndent(schema, "", " ")
					os.WriteFile(baseDir+"/"+api.Name+"/schema_definition.json", bytes, 0644)

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
