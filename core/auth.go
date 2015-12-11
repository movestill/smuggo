package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/go-oauth/oauth"
	"github.com/skratchdot/open-golang/open"
	"io/ioutil"
	"os"
)

const (
	oauthOrigin       = "https://secure.smugmug.com"
	oauthAuthorize    = oauthOrigin + "/services/oauth/1.0a/authorize"
	oauthRequestToken = oauthOrigin + "/services/oauth/1.0a/getRequestToken"
	oauthAccessToken  = oauthOrigin + "/services/oauth/1.0a/getAccessToken"
)

var oauthClient oauth.Client

func authInit() {
	apiToken, err := loadToken(apiTokenFile)
	if err != nil {
		fmt.Println("Error reading " + apiTokenFile + ": " + err.Error())
		os.Exit(1)
	}

	oauthClient = oauth.Client{
		TemporaryCredentialRequestURI: oauthRequestToken,
		ResourceOwnerAuthorizationURI: oauthAuthorize,
		TokenRequestURI:               oauthAccessToken,
		Credentials:                   *apiToken,
	}
}

func auth() {
	tempCred, err := beginAuth()
	if err != nil {
		return
	}

	fmt.Print("Enter your verification code: ")
	var verifyCode string
	if _, err := fmt.Scanln(&verifyCode); err != nil {
		fmt.Println("Error reading verification code " + err.Error())
		return
	}

	accessCred, err := completeAuth(tempCred, verifyCode)
	if err != nil {
		return
	}

	if err := storeAccessToken(accessCred, userTokenFile); err != nil {
		fmt.Println("Error saving access token: " + err.Error())
		return
	}

	fmt.Println("smuggo authorized.  Access token saved to " + userTokenFile)
}

func beginAuth() (*oauth.Credentials, error) {
	tempCred, err := oauthClient.RequestTemporaryCredentials(nil, "oob", nil)
	if err != nil {
		fmt.Print("Error getting temp credentials: " + err.Error())
		return nil, err
	}
	url := oauthAuthorize + "?Access=Full&Permissions=All&oauth_token=" + tempCred.Token
	open.Start(url)
	fmt.Println("Opening browser with " + url)
	return tempCred, nil
}

func completeAuth(tempCred *oauth.Credentials, verifyCode string) (*oauth.Credentials, error) {
	credentials, _, err := oauthClient.RequestToken(nil, tempCred, verifyCode)
	if err != nil {
		fmt.Println("Error getting token: " + err.Error())
		return nil, err
	}
	return credentials, nil
}

func storeAccessToken(accessCred *oauth.Credentials, filename string) error {
	bytes, err := json.MarshalIndent(*accessCred, "", "    ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, bytes, 0600); err != nil {
		return err
	}

	return nil
}
