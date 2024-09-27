package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

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

func awsCleanLocal(flags *AwsFlags) error {
	var baseDir = "src/main/aws"
	os.RemoveAll(baseDir)
	return nil
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
						outputType := "JSON"
						specType := "OAS30"
						apiExport, exportErr := client.ExportApi(context.TODO(), &apigatewayv2.ExportApiInput{
							ApiId:         api.ApiId,
							OutputType:    &outputType,
							Specification: &specType,
						})

						if exportErr != nil {
							fmt.Println(exportErr)
						}

						bytes, _ := json.MarshalIndent(api, "", " ")

						newName := strings.ReplaceAll(strings.ToLower(*api.Name), " ", "-")

						var re = regexp.MustCompile(`(-v\d+)$`)
						newName2 := re.ReplaceAllString(newName, "")

						// newName2 := newName
						// if *api.Version != "" {
						// 	newName2 += "-v" + strings.Split(*api.Version, ".")[0]
						// }

						fmt.Println(newName2)
						fmt.Println(newName)

						os.RemoveAll(baseDir + "/" + newName)
						os.MkdirAll(baseDir+"/"+newName2, 0755)
						os.WriteFile(baseDir+"/"+newName2+"/"+newName+".json", bytes, 0644)
						if apiExport != nil && apiExport.Body != nil {
							os.WriteFile(baseDir+"/"+newName2+"/"+newName+"-oas.json", apiExport.Body, 0644)
						}
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

	entries, err := os.ReadDir(awsBaseDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Offramping AWS API Gateway APIs to general...")

	for _, e := range entries {
		if flags.ApiName == "" || flags.ApiName == e.Name() {
			fmt.Println(e.Name())

			// read all files
			fileEntries, _ := os.ReadDir(awsBaseDir + "/" + e.Name())
			for _, f := range fileEntries {
				if !strings.HasSuffix(f.Name(), "-oas.json") && !strings.HasSuffix(f.Name(), "-oas-definition.json") {
					var awsApi types.Api
					apiFile, err := os.Open(awsBaseDir + "/" + e.Name() + "/" + f.Name())
					if err != nil {
						log.Fatal(err)
					} else {
						byteValue, _ := io.ReadAll(apiFile)
						json.Unmarshal(byteValue, &awsApi)
					}
					defer apiFile.Close()

					if *awsApi.Name != "" {
						var generalApi GeneralApi
						baseName := strings.ReplaceAll(strings.ToLower(*awsApi.Name), " ", "-")
						generalApi.Name = baseName + "-aws"
						generalApi.DisplayName = *awsApi.Name
						generalApi.Description = *awsApi.Description
						generalApi.Version = *awsApi.Version
						generalApi.GatewayUrl = *awsApi.ApiEndpoint
						generalApi.PlatformId = "aws-api-gateway"
						generalApi.PlatformName = "AWS API Gateway"
						generalApi.PlatformResourceUri = "https://" + flags.Region + ".console.aws.amazon.com/apigateway/main/apis?api=" + *awsApi.ApiId

						bytes, _ := json.MarshalIndent(generalApi, "", " ")
						//os.RemoveAll(baseDir + "/" + generalApi.Name)
						os.MkdirAll(baseDir+"/"+e.Name(), 0755)

						os.WriteFile(baseDir+"/"+e.Name()+"/"+e.Name()+".json", bytes, 0644)
						os.WriteFile(baseDir+"/"+e.Name()+"/"+generalApi.Name+".json", bytes, 0644)

						schemaFile, err := os.Open(awsBaseDir + "/" + e.Name() + "/" + baseName + "-oas.json")
						if err == nil {
							// we have an api spec, copy it over
							byteValue, _ := io.ReadAll(schemaFile)
							os.WriteFile(baseDir+"/"+e.Name()+"/"+generalApi.Name+"-oas.json", byteValue, 0644)
						}
					}
				}
			}
		}
	}

	return nil
}
