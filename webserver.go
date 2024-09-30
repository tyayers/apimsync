package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/go-chi/chi/v5"

	_ "github.com/danielgtaylor/huma/v2/formats/cbor"
)

type WebServerFlags struct {
	Port int `name:"port" description:"The port to listen on." help:"The port to listen on." default:"8080"`
}

type ApimStatus struct {
	Body struct {
		ApigeeStatus PlatformStatus `json:"apigee"`
		ApiHubStatus PlatformStatus `json:"apihub"`
		AzureStatus  PlatformStatus `json:"azure"`
		AwsStatus    PlatformStatus `json:"aws"`
	}
}

type ApimOfframpInput struct {
	Body struct {
		Offramp string `json:"offramp" enum:"azure,aws" doc:"The APIM platform to offramp the APIs from."`
		OnlyNew bool   `json:"onlyNew" doc:"Default is false, only offramp new APIs. Set to false to offramp all APIs."`
	}
}

type ApimOfframpOutput struct {
	Body struct {
		Result  bool     `json:"result" example:"true" doc:"The result of the offramp operation."`
		Apis    []string `json:"apis" example:"[\"api1\", \"api2\"]" doc:"The names of the APIs that were offramped."`
		Message string   `json:"message" example:"Offramp successful!" doc:"The result of the offramp operation."`
	}
}

type ApimOnrampInput struct {
	Body struct {
		Onramp string `json:"onramp" enum:"apihub" doc:"The API platform to onramp the APIs to."`
	}
}

type ApimOnrampOutput struct {
	Body struct {
		Result  bool   `json:"result" example:"true" doc:"The result of the onramp operation."`
		Message string `json:"message" example:"Onramp successful!" doc:"The result of the onramp operation."`
	}
}

type ApimSyncInput struct {
	Body struct {
		Offramp string `json:"offramp" enum:"azure,aws" doc:"The APIM platform to offramp the APIs from."`
		Onramp  string `json:"onramp" enum:"apihub" doc:"The APIM platform to onramp the APIs to."`
	}
}

type ApimSyncOutput struct {
	Body struct {
		Result  bool   `json:"result" example:"true" doc:"The result of the sync operation."`
		Message string `json:"message" example:"Sync successful!" doc:"The result of the sync operation."`
	}
}

func webServerStart(flags *WebServerFlags) error {
	// Create a CLI app which takes a port option.
	cli := humacli.New(func(hooks humacli.Hooks, options *WebServerFlags) {
		// Create a new router & API
		router := chi.NewMux()
		api := humachi.New(router, huma.DefaultConfig("Apimsync API", "0.1.6"))

		// Add the operation handler to the API.
		huma.Get(api, "/v1/apim/status", apimStatus)
		huma.Post(api, "/v1/apim/offramp", apimOfframp)
		huma.Post(api, "/v1/apim/onramp", apimOnramp)
		huma.Post(api, "/v1/apim/sync", apimSync)

		hooks.OnStart(func() {
			http.ListenAndServe(fmt.Sprintf(":%d", options.Port), router)
		})
	})

	cli.Run()
	return nil
}

func apimStatus(ctx context.Context, input *struct{}) (*ApimStatus, error) {
	var status ApimStatus
	apigeeFlags := ApigeeFlags{Project: os.Getenv("APIGEE_PROJECT"), Region: os.Getenv("APIGEE_REGION")}
	azureFlags := AzureFlags{Subscription: os.Getenv("AZURE_SUBSCRIPTION_ID"), ResourceGroup: os.Getenv("AZURE_RESOURCE_GROUP"), ServiceName: os.Getenv("AZURE_SERVICE_NAME")}
	awsFlags := AwsFlags{Region: os.Getenv("AWS_REGION"), AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"), AccessSecret: os.Getenv("AWS_SECRET_ACCESS_KEY")}
	status.Body.ApigeeStatus = apigeeStatus(&apigeeFlags)
	status.Body.ApiHubStatus = apiHubStatus(&apigeeFlags)
	status.Body.AzureStatus = azureStatus(&azureFlags)
	status.Body.AwsStatus = awsStatus(&awsFlags)

	return &status, nil
}

func apimOfframp(ctx context.Context, input *ApimOfframpInput) (*ApimOfframpOutput, error) {
	var result ApimOfframpOutput

	azureFlags := AzureFlags{Subscription: os.Getenv("AZURE_SUBSCRIPTION_ID"), ResourceGroup: os.Getenv("AZURE_RESOURCE_GROUP"), ServiceName: os.Getenv("AZURE_SERVICE_NAME")}
	azureFlags.OnlyNew = input.Body.OnlyNew
	awsFlags := AwsFlags{Region: os.Getenv("AWS_REGION"), AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"), AccessSecret: os.Getenv("AWS_SECRET_ACCESS_KEY")}
	awsFlags.OnlyNew = input.Body.OnlyNew

	if input.Body.Offramp == "azure" {
		azureServiceExport(&azureFlags)
		result.Body.Apis, _ = azureExport(&azureFlags)
		azureOfframp(&azureFlags)
		result.Body.Result = true
	} else if input.Body.Offramp == "aws" {
		result.Body.Apis, _ = awsExport(&awsFlags)
		awsOfframp(&awsFlags)
		result.Body.Result = true
	}

	if result.Body.Result {
		if result.Body.Apis == nil {
			result.Body.Apis = []string{}
		}
		result.Body.Message = strconv.Itoa(len(result.Body.Apis)) + " API(s) offramped."
	}

	return &result, nil
}

func apimOnramp(ctx context.Context, input *ApimOnrampInput) (*ApimOnrampOutput, error) {
	var result ApimOnrampOutput

	apigeeFlags := ApigeeFlags{Project: os.Getenv("APIGEE_PROJECT"), Region: os.Getenv("APIGEE_REGION")}

	if input.Body.Onramp == "apihub" {
		apiHubOnramp(&apigeeFlags)
		apiHubImport(&apigeeFlags)
	}

	result.Body.Result = true
	result.Body.Message = "Onramp to " + input.Body.Onramp + " successful!"
	return &result, nil
}

func apimSync(ctx context.Context, input *ApimSyncInput) (*ApimSyncOutput, error) {
	var result ApimSyncOutput

	apigeeFlags := ApigeeFlags{Project: os.Getenv("APIGEE_PROJECT"), Region: os.Getenv("APIGEE_REGION")}
	azureFlags := AzureFlags{Subscription: os.Getenv("AZURE_SUBSCRIPTION_ID"), ResourceGroup: os.Getenv("AZURE_RESOURCE_GROUP"), ServiceName: os.Getenv("AZURE_SERVICE_NAME")}
	awsFlags := AwsFlags{Region: os.Getenv("AWS_REGION"), AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"), AccessSecret: os.Getenv("AWS_SECRET_ACCESS_KEY")}

	if input.Body.Offramp == "azure" {
		azureServiceExport(&azureFlags)
		azureExport(&azureFlags)
		azureOfframp(&azureFlags)
	} else if input.Body.Offramp == "aws" {
		awsExport(&awsFlags)
		awsOfframp(&awsFlags)
	}

	if input.Body.Onramp == "apihub" {
		apiHubOnramp(&apigeeFlags)
		apiHubImport(&apigeeFlags)
	}

	result.Body.Result = true
	result.Body.Message = "Sync from " + input.Body.Offramp + " to " + input.Body.Onramp + " successful!"
	return &result, nil
}
