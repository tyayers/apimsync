package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
)

type AwsApis struct {
	Items []AwsApi `json:"items"`
}

type AwsApi struct {
	ApiId                     string               `json:"apiId"`
	Name                      string               `json:"name"`
	Description               string               `json:"description"`
	ProtocolType              string               `json:"protocolType"`
	RouteSelectionExpression  string               `json:"routeSelectionExpression"`
	ApiKeySelectionExpression string               `json:"apiKeySelectionExpression"`
	DisableSchemaValidation   bool                 `json:"disableSchemaValidation"`
	Warnings                  []string             `json:"warnings"`
	ImportInfo                []string             `json:"importInfo"`
	ApiEndpoint               string               `json:"apiEndpoint"`
	ApiGatewayManaged         bool                 `json:"apiGatewayManaged"`
	CreatedDate               string               `json:"createdDate"`
	Tags                      map[string]string    `json:"tags"`
	DisableExecuteApiEndpoint bool                 `json:"disableExecuteApiEndpoint"`
	CorsConfiguration         AwsCorsConfiguration `json:"corsConfiguration"`
}

type AwsCorsConfiguration struct {
	AllowCredentials bool     `json:"allowCredentials"`
	AllowHeaders     []string `json:"allowHeaders"`
	AllowMethods     []string `json:"allowMethods"`
	AllowOrigins     []string `json:"allowOrigins"`
	ExposeHeaders    []string `json:"exposeHeaders"`
	MaxAge           int32    `json:"maxAge"`
}

type AwsFlags struct {
	AccessKey    string `name:"accessKey" description:"The AWS access key to use to authenticate with AWS."`
	AccessSecret string `name:"accessSecret" description:"The AWS secret key to use to authenticate with AWS."`
	Region       string `name:"region" description:"The AWS region of the API Gateway."`
	ApiName      string `name:"api" description:"A specific Azure API Management API."`
}

func awsStatus(flags *AwsFlags) PlatformStatus {
	var status PlatformStatus

	if flags.Region == "" {
		flags.Region = os.Getenv("AWS_REGION")
	}
	if flags.AccessKey != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", flags.AccessKey)
	}
	if flags.AccessSecret != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", flags.AccessSecret)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(flags.Region))
	if err != nil {
		log.Fatal(err)
	}

	client := apigatewayv2.NewFromConfig(cfg)

	if client != nil {
		apis, _ := client.GetApis(context.TODO(), &apigatewayv2.GetApisInput{})

		if apis != nil {
			status.Connected = true
			status.Message = "Connected to Aws, " + strconv.Itoa(len(apis.Items)) + " API(s) found ."
		}
	}

	return status
}

func awsExport(flags *AwsFlags) error {
	var baseDir = "src/main/aws/apiproxies"
	if flags.Region == "" {
		flags.Region = os.Getenv("AWS_REGION")
		if flags.Region == "" {
			fmt.Println("No region given, cannot export AWS APIs.")
			return nil
		}
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(flags.Region))
	if err != nil {
		log.Fatal(err)
	}

	client := apigatewayv2.NewFromConfig(cfg)

	if client != nil {
		fmt.Println("Exporting AWS APIs for region " + flags.Region + "...")

		apis, _ := client.GetApis(context.TODO(), &apigatewayv2.GetApisInput{})

		if apis != nil {
			if len(apis.Items) > 0 {
				for _, api := range apis.Items {
					if flags.ApiName == "" || flags.ApiName == *api.Name {
						fmt.Println("Exporting " + *api.Name + "...")
						bytes, _ := json.MarshalIndent(api, "", " ")
						os.RemoveAll(baseDir + "/" + *api.Name)
						os.MkdirAll(baseDir+"/"+*api.Name, 0755)
						os.WriteFile(baseDir+"/"+*api.Name+"/"+*api.Name+".json", bytes, 0644)
					}
				}
			} else {
				fmt.Println("No AWS APIs found in region " + flags.Region + ", cannot export APIs.")
				return nil
			}
		} else {
			fmt.Println("No valid APIs found in region " + flags.Region + ", cannot export APIs.")
			return nil
		}
	} else {
		fmt.Println("AWS client could not be created, cannot export APIs.")
		return nil
	}

	return nil
}

func awsOfframp(flags *AwsFlags) error {

	awsBaseDir := "src/main/aws/apiproxies"
	baseDir := "src/main/general/apiproxies"

	if flags.Region == "" {
		flags.Region = os.Getenv("AWS_REGION")
		if flags.Region == "" {
			fmt.Println("No region given, cannot export AWS APIs.")
			return nil
		}
	}

	entries, err := os.ReadDir(awsBaseDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Offramping AWS API Gateway APIs to general...")

	for _, e := range entries {
		if flags.ApiName == "" || flags.ApiName == e.Name() {
			fmt.Println(e.Name())
			var awsApi types.Api
			apiFile, err := os.Open(awsBaseDir + "/" + e.Name() + "/" + e.Name() + ".json")
			if err != nil {
				log.Fatal(err)
			} else {
				byteValue, _ := io.ReadAll(apiFile)
				json.Unmarshal(byteValue, &awsApi)
			}
			defer apiFile.Close()

			if *awsApi.Name != "" {
				var generalApi GeneralApi
				generalApi.Name = *awsApi.Name
				generalApi.DisplayName = *awsApi.Name
				generalApi.Description = *awsApi.Description
				generalApi.Version = *awsApi.Version
				generalApi.GatewayUrl = *awsApi.ApiEndpoint
				generalApi.PlatformId = "aws"
				generalApi.PlatformName = "AWS API Gateway"

				bytes, _ := json.MarshalIndent(generalApi, "", " ")
				os.RemoveAll(baseDir + "/" + generalApi.Name)
				os.MkdirAll(baseDir+"/"+generalApi.Name, 0755)

				os.WriteFile(baseDir+"/"+generalApi.Name+"/"+generalApi.Name+".json", bytes, 0644)
			}
		}
	}

	return nil
}
