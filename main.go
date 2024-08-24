package main

import (
	"log"

	"github.com/leaanthony/clir"
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
	BasePath            string `json:"basePath"`
	PlatformId          string `json:"platformId"`
	PlatformName        string `json:"platformName"`
	PlatformResourceUri string `json:"platformResourceUri"`
}

type PlatformStatus struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
}

func main() {
	// Create new cli
	cli := clir.NewCli("apimsync", "A syncing tool between API platforms", "v0.0.1")

	webServerCommand := cli.NewSubCommand("ws", "Functions for the web server.")
	webServerCommand.NewSubCommandFunction("start", "Start a web server to listen for commands.", webServerStart)

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
