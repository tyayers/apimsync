package main

import (
	"encoding/json"
	"io"
	"os"
	"regexp"
)

func generalCleanLocal(flags *GeneralFlags) error {
	var baseDir = "src/main/general"
	os.RemoveAll(baseDir)
	return nil
}

func writeGeneralApi(name string, generalApi GeneralApi) error {
	baseDir := "src/main/general/apiproxies"

	generalApi.Name = name
	var re = regexp.MustCompile(` v\d+`)
	generalApi.DisplayName = re.ReplaceAllString(generalApi.DisplayName, "")

	newBytes, _ := json.MarshalIndent(generalApi, "", "  ")

	apiFile, err := os.Open(baseDir + "/" + name + "/" + name + ".json")
	if err == nil {
		byteValue, _ := io.ReadAll(apiFile)

		// only overwrite if new file is larger, meaning it has more information
		if len(newBytes) > len(byteValue) {
			os.WriteFile(baseDir+"/"+name+"/"+name+".json", newBytes, 0644)
		}
	} else {
		os.WriteFile(baseDir+"/"+name+"/"+name+".json", newBytes, 0644)
	}

	return nil
}
