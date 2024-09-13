package main

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

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

type HubApiVersionExtended struct {
	HubApiVersion
	BaseApiName string `json:"baseApiName"`
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

func apiHubStatus(flags *ApigeeFlags) PlatformStatus {
	var status PlatformStatus
	if flags.Project == "" {
		status.Connected = false
		status.Message = "No project given, cannot connect to API Hub."
		return status
	} else if flags.Region == "" {
		status.Connected = false
		status.Message = "No region given, cannot connect to API Hub."
		return status
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
	req, _ := http.NewRequest(http.MethodGet, "https://apihub.googleapis.com/v1/projects/"+flags.Project+"/locations/"+flags.Region+"/apis", nil)
	req.Header.Add("Authorization", "Bearer "+flags.Token)

	var apis HubApis
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			json.Unmarshal(body, &apis)
		}

		if resp.StatusCode == 200 {
			status.Connected = true
			status.Message = "Connected to API Hub, " + strconv.Itoa(len(apis.Apis)) + " APIs found in project " + flags.Project + " and region " + flags.Region + "."
		} else {
			status.Connected = false
			status.Message = resp.Status
		}
	} else {
		status.Connected = false
		status.Message = err.Error()
	}

	return status
}

func apiHubOnramp(flags *ApigeeFlags) error {
	generalBaseDir := "src/main/general/apiproxies"
	baseDir := "src/main/apihub/apiproxies"

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

	// First index all root APIs
	var rootApis map[string]string = make(map[string]string)
	for _, e := range entries {
		var generalApi GeneralApi
		apiFile, err := os.Open(generalBaseDir + "/" + e.Name() + "/" + e.Name() + ".json")
		if err != nil {
			log.Fatal(err)
		} else {
			byteValue, _ := io.ReadAll(apiFile)
			json.Unmarshal(byteValue, &generalApi)
		}
		defer apiFile.Close()

		if generalApi.Version == "" {
			// This is a root API (version 1)
			rootApis[generalApi.BasePath] = generalApi.Name
		}
	}

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

				var apiName = generalApi.Name
				rootName, ok := rootApis[generalApi.BasePath]
				if ok {
					apiName = rootName
				}

				// create API, if it is a root API (not just a version)
				if generalApi.Version == "" {
					var hubApi HubApi
					hubApi.Name = "projects/" + flags.Project + "/locations/" + flags.Region + "/apis/" + apiName
					hubApi.DisplayName = generalApi.DisplayName
					hubApi.Description = generalApi.Description
					hubApi.Documentation.ExternalUri = generalApi.DocumentationUrl
					hubApi.Owner.DisplayName = generalApi.OwnerName
					hubApi.Owner.Email = generalApi.OwnerEmail
					bytes, _ := json.MarshalIndent(hubApi, "", "  ")
					os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+".json", bytes, 0644)
				}

				// create deployment
				var hubApiDeployment HubApiDeployment
				hubApiDeployment.Name = "projects/" + flags.Project + "/locations/" + flags.Region + "/deployments/" + generalApi.Name
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
				bytes, _ := json.MarshalIndent(hubApiDeployment, "", "  ")
				os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+"-deployment.json", bytes, 0644)

				// create API version
				// we use the extended version here to include the BaseApiName that we use in apihub to connect the version to a base api
				var hubApiVersion HubApiVersionExtended
				hubApiVersion.BaseApiName = apiName
				hubApiVersion.Name = "projects/" + flags.Project + "/locations/" + flags.Region + "/apis/" + apiName + "/versions/" + generalApi.Name
				hubApiVersion.DisplayName = generalApi.DisplayName
				hubApiVersion.Description = generalApi.Description
				hubApiVersion.Documentation.ExternalUri = generalApi.DocumentationUrl
				hubApiVersion.Deployments = append(hubApiVersion.Deployments, hubApiDeployment.Name)
				bytes, _ = json.MarshalIndent(hubApiVersion, "", "  ")
				os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+"-version.json", bytes, 0644)

				// create API spec, if available
				b, err := os.ReadFile(generalBaseDir + "/" + generalApi.Name + "/openapi.json")
				if err == nil {
					// we have a spec file
					var hubApiVersionSpec HubApiVersionSpec
					hubApiVersionSpec.Name = "projects/" + flags.Project + "/locations/" + flags.Region + "/apis/" + apiName + "/versions/" + generalApi.Name + "/specs/" + generalApi.Name
					hubApiVersionSpec.DisplayName = generalApi.DisplayName
					apiSpecType := HubAttributeValue{Id: "openapi", DisplayName: "OpenAPI Spec", Description: "OpenAPI Spec", Immutable: true}
					hubApiVersionSpec.SpecType.EnumValues.Values = append(hubApiVersionSpec.SpecType.EnumValues.Values, apiSpecType)
					hubApiVersionSpec.Contents.MimeType = "application/json"
					hubApiVersionSpec.Contents.Contents = b64.StdEncoding.EncodeToString(b)
					hubApiVersionSpec.Documentation.ExternalUri = generalApi.DocumentationUrl
					bytes, _ = json.MarshalIndent(hubApiVersionSpec, "", "  ")
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
	var baseDir = "src/main/apihub/apiproxies"
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
				// Create API
				apiFile, err := os.Open(baseDir + "/" + e.Name() + "/" + e.Name() + ".json")
				if err == nil {
					var hubApi HubApi
					byteValue, _ := io.ReadAll(apiFile)
					json.Unmarshal(byteValue, &hubApi)

					requestBody := bytes.NewBuffer(byteValue)
					r, _ := http.NewRequest(http.MethodPost, "https://apihub.googleapis.com/v1/projects/"+flags.Project+"/locations/"+flags.Region+"/apis?apiId="+e.Name(), requestBody)
					r.Header.Add("Content-Type", "application/json")
					r.Header.Add("Authorization", "Bearer "+flags.Token)
					client := &http.Client{}
					fmt.Println("Creating API " + e.Name() + "...")
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
					deploymentUrl := "https://apihub.googleapis.com/v1/projects/" + flags.Project + "/locations/" + flags.Region + "/deployments?deploymentId=" + e.Name()
					r, _ := http.NewRequest(http.MethodPost, deploymentUrl, requestBody)
					r.Header.Add("Content-Type", "application/json")
					r.Header.Add("Authorization", "Bearer "+flags.Token)
					client := &http.Client{}
					fmt.Println("Creating deployment " + e.Name() + "...")
					resp, _ := client.Do(r)

					if resp.StatusCode != 200 {
						fmt.Println("  >> Error creating deployment " + e.Name() + ": " + resp.Status)
						defer resp.Body.Close()
						//Read the response body
						respBody, _ := io.ReadAll(resp.Body)
						sb := string(respBody)
						fmt.Println(sb)
					}
				}
				defer deploymentFile.Close()

				// Create API Version
				baseApiName := e.Name()
				versionFile, err := os.Open(baseDir + "/" + e.Name() + "/" + e.Name() + "-version.json")
				if err == nil {
					var apiVersionExtended HubApiVersionExtended
					byteValue, _ := io.ReadAll(versionFile)
					json.Unmarshal(byteValue, &apiVersionExtended)
					var apiVersion HubApiVersion = apiVersionExtended.HubApiVersion
					bodyBytes, _ := json.Marshal(apiVersion)
					requestBody := bytes.NewBuffer(bodyBytes)

					baseApiName = apiVersionExtended.BaseApiName
					versionUrl := "https://apihub.googleapis.com/v1/projects/" + flags.Project + "/locations/" + flags.Region + "/apis/" + baseApiName + "/versions?versionId=" + e.Name()
					r, _ := http.NewRequest(http.MethodPost, versionUrl, requestBody)
					r.Header.Add("Content-Type", "application/json")
					r.Header.Add("Authorization", "Bearer "+flags.Token)
					client := &http.Client{}
					fmt.Println("Creating API version " + e.Name() + "...")
					resp, _ := client.Do(r)

					if resp.StatusCode != 200 {
						fmt.Println("  >> Error creating version " + e.Name() + ": " + resp.Status)
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

					versionUrl := "https://apihub.googleapis.com/v1/projects/" + flags.Project + "/locations/" + flags.Region + "/apis/" + baseApiName + "/versions/" + e.Name() + "/specs?specId=" + e.Name()
					r, _ := http.NewRequest(http.MethodPost, versionUrl, requestBody)
					r.Header.Add("Content-Type", "application/json")
					r.Header.Add("Authorization", "Bearer "+flags.Token)
					client := &http.Client{}
					fmt.Println("Creating API version spec " + e.Name() + "...")
					resp, _ := client.Do(r)

					if resp.StatusCode != 200 {
						fmt.Println("  >> Error deploying version spec " + e.Name() + ": " + resp.Status)
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
