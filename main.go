package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	var token *oauth2.Token
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}

	ctx := context.Background()
	credentials, err := google.FindDefaultCredentials(ctx, scopes...)

	if err == nil {
		token, err = credentials.TokenSource.Token()

		if err == nil {
			getApis("apigee-test38", token.AccessToken)
		}
	}
}

func getApis(org string, token string) {
	req, _ := http.NewRequest(http.MethodGet, "https://apigee.googleapis.com/v1/organizations/"+org+"/apis?includeRevisions=true", nil)
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			fmt.Println(string(body))
		}
	}

}
