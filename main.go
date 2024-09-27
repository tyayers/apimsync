package main

import (
	"log"
	"os"

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

type GeneralFlags struct {
	ApiName string `name:"api" description:"A specific Azure API Management API."`
}

func main() {
	// Create new cli
	cli := clir.NewCli("apimsync", "A syncing tool for API & integration platforms", "v0.1.5")

	generalCommand := cli.NewSubCommand("general", "Functions for general offramped APIs.")
	generalApisCommand := generalCommand.NewSubCommand("apis", "Functions for General API resources.")
	generalApisCommand.NewSubCommandFunction("cleanlocal", "Removes all APIs from offramped general definitions in local storage.", generalCleanLocal)

	webServerCommand := cli.NewSubCommand("ws", "Functions for the web server.")
	webServerCommand.NewSubCommandFunction("start", "Start a web server to listen for commands.", webServerStart)

	apigeeCommand := cli.NewSubCommand("apigee", "Functions for Apigee.")
	apigeeApisCommand := apigeeCommand.NewSubCommand("apis", "Functions for Apigee API resources.")
	apigeeApisCommand.NewSubCommandFunction("export", "Exports Apigee APIs from a given project.", apigeeExport)
	apigeeApisCommand.NewSubCommandFunction("import", "Imports APIs to an Apigee project.", apigeeImport)
	apigeeApisCommand.NewSubCommandFunction("clean", "Removes all of the Apigee APIs from a given project.", apigeeClean)
	apigeeTestCommand := apigeeCommand.NewSubCommand("test", "Local test commands.")
	apigeeTestCommand.NewSubCommandFunction("init", "Initializes local test data for an environment.", initApigeeTest)

	apiHubCommand := cli.NewSubCommand("apihub", "Functions for Apigee API Hub.")
	apiHubApisCommand := apiHubCommand.NewSubCommand("apis", "Functions for API Hub API resources.")
	apiHubApisCommand.NewSubCommandFunction("onramp", "Onramps APIs from general to API Hub.", apiHubOnramp)
	apiHubApisCommand.NewSubCommandFunction("import", "Imports APIs to API Hub.", apiHubImport)
	apiHubApisCommand.NewSubCommandFunction("clean", "Removes all APIs from API Hub.", apiHubClean)
	apiHubApisCommand.NewSubCommandFunction("cleanlocal", "Removes all API Hub APIs from local storage.", apiHubCleanLocal)

	azureCommand := cli.NewSubCommand("azure", "Functions for Azure API Management.")
	azureCommand.NewSubCommandFunction("export", "Exports Azure API Management service info.", azureServiceExport)
	azureApisCommand := azureCommand.NewSubCommand("apis", "Functions for Azure API Management API resources.")
	azureApisCommand.NewSubCommandFunction("export", "Exports Azure API Management APIs.", azureExport)
	azureApisCommand.NewSubCommandFunction("offramp", "Migrates Azure API Management APIs out to general.", azureOfframp)
	azureApisCommand.NewSubCommandFunction("cleanlocal", "Removes all exported Azure APIs from local storage.", azureCleanLocal)

	awsCommand := cli.NewSubCommand("aws", "Functions for AWS API Gateway.")
	awsApisCommand := awsCommand.NewSubCommand("apis", "Functions for AWS API Gateway API resources.")
	awsApisCommand.NewSubCommandFunction("export", "Exports AWS API Gateway APIs.", awsExport)
	awsApisCommand.NewSubCommandFunction("offramp", "Offramp AWS API Gateway APIs.", awsOfframp)
	awsApisCommand.NewSubCommandFunction("cleanlocal", "Removes all exported AWS APIs from local storage.", awsCleanLocal)

	err := cli.Run()

	if err != nil {
		// We had an error
		log.Fatal(err)
	}
}

func generalCleanLocal(flags *GeneralFlags) error {
	var baseDir = "src/main/general"
	os.RemoveAll(baseDir)
	return nil
}
