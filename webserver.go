package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

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
	}
}

type ApimSyncInput struct {
	Body struct {
		Offramp string `example:"azure"`
		Onramp  string `example:"apihub"`
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
		api := humachi.New(router, huma.DefaultConfig("Apimsync API", "0.1.1"))

		// Add the operation handler to the API.
		huma.Get(api, "/v1/apim/status", apimStatus)

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
	status.Body.ApigeeStatus = apigeeStatus(&apigeeFlags)
	status.Body.ApiHubStatus = apiHubStatus(&apigeeFlags)
	status.Body.AzureStatus = azureStatus(&azureFlags)

	return &status, nil
}

func apimSync(ctx context.Context, input *ApimSyncInput) (*ApimSyncOutput, error) {
	var result ApimSyncOutput

	apigeeFlags := ApigeeFlags{Project: os.Getenv("APIGEE_PROJECT"), Region: os.Getenv("APIGEE_REGION")}
	azureFlags := AzureFlags{Subscription: os.Getenv("AZURE_SUBSCRIPTION_ID"), ResourceGroup: os.Getenv("AZURE_RESOURCE_GROUP"), ServiceName: os.Getenv("AZURE_SERVICE_NAME")}

	if input.Body.Offramp == "azure" {
		azureExport(&azureFlags)
		azureOfframp(&azureFlags)
	}

	if input.Body.Onramp == "apihub" {
		apiHubOnramp(&apigeeFlags)
		apiHubImport(&apigeeFlags)
	}

	result.Body.Result = true
	result.Body.Message = "Sync from " + input.Body.Offramp + " to " + input.Body.Onramp + " successful!"
	return &result, nil
}