package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/go-oauth/oauth"
	"github.com/skratchdot/open-golang/open"
	"io/ioutil"
	"os"
	"os/user"
)

const (
	oauthOrigin       = "https://secure.smugmug.com"
	oauthAuthorize    = oauthOrigin + "/services/oauth/1.0a/authorize"
	oauthRequestToken = oauthOrigin + "/services/oauth/1.0a/getRequestToken"
	oauthAccessToken  = oauthOrigin + "/services/oauth/1.0a/getAccessToken"
)

// Handles all OAuth stuff.
var oauthClient oauth.Client

// Save the home directory for storing JSON files, later.
var userHomeDir string

// This is appended to userHomeDir.
const smuggoDir = "/.smuggo/"

func authInit() {
	user, err := user.Current()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	userHomeDir = user.HomeDir

	apiToken, err := loadToken(userHomeDir + smuggoDir + apiTokenFile)
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

// Authorize smuggo for the user's account.
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

	fullPathTokenFile := userHomeDir + smuggoDir + userTokenFile

	if err := storeAccessToken(accessCred, fullPathTokenFile); err != nil {
		fmt.Println("Error saving access token: " + err.Error())
		return
	}

	fmt.Println("smuggo authorized.  Access token saved to " + fullPathTokenFile)
}

// Start the auth process.
func beginAuth() (*oauth.Credentials, error) {
	tempCred, err := oauthClient.RequestTemporaryCredentials(nil, "oob", nil)
	if err != nil {
		fmt.Print("Error getting temp credentials: " + err.Error())
		return nil, err
	}
	url := oauthAuthorize + "?Access=Full&Permissions=Modify&oauth_token=" + tempCred.Token
	open.Start(url)
	fmt.Println("Opening browser with " + url)
	return tempCred, nil
}

// Send user's verification code back to SmugMug and get a permanent OAuth
// token.
func completeAuth(tempCred *oauth.Credentials, verifyCode string) (*oauth.Credentials, error) {
	credentials, _, err := oauthClient.RequestToken(nil, tempCred, verifyCode)
	if err != nil {
		fmt.Println("Error getting token: " + err.Error())
		return nil, err
	}
	return credentials, nil
}

// Write OAuth token to disk.
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

// Get the user token from the appropriate location.
func loadUserToken() (*oauth.Credentials, error) {
	fullPathTokenFile := userHomeDir + smuggoDir + userTokenFile
	return loadToken(fullPathTokenFile)
}
