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

type ApigeeFlags struct {
	Project string `name:"project" description:"The Google Cloud project that Apigee is running in."`
	Token   string `name:"token" description:"The Google access token to call Apigee with."`
	ApiName string `name:"api" description:"A specific Apigee API."`
}

func main() {
	// Create new cli
	cli := clir.NewCli("apimsync", "A syncing tool between API platforms", "v0.0.1")

	apigeeCommand := cli.NewSubCommand("apigee", "Functions for Apigee APIs.")
	apigeeCommand.NewSubCommandFunction("export", "Exports Apigee APIs from a given project.", apigeeExport)
	apigeeCommand.NewSubCommandFunction("import", "Imports APIs to an Apigee project.", apigeeImport)
	apigeeCommand.NewSubCommandFunction("clean", "Removes all of the Apigee APIs from a given project.", apigeeClean)
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
				}
			}
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
	//r, _ := http.NewRequest(http.MethodPost, "https://testty.free.beeceptor.com", body)
	r.Header.Add("Content-Type", writer.FormDataContentType())
	r.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(r)

	if resp.StatusCode != 200 {
		fmt.Println("Error creating Apigee API: " + resp.Status)
	}

	return err
}
