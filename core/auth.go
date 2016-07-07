package main

import (
	"encoding/json"
	"fmt"
	"go-oauth/oauth"
	"io/ioutil"
	"log"
	"open-golang/open"
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

// Get and store the user's home directory.
func getUserHomeDir() string {
	user, err := user.Current()
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	userHomeDir = user.HomeDir

	return userHomeDir
}

func authInit() {
	getUserHomeDir()

	err := os.MkdirAll(userHomeDir+smuggoDir, os.ModeDir|0700)
	if err != nil {
		log.Println("Could not create smuggo data folder: " + err.Error())
		os.Exit(1)
	}

	apiToken, err := loadToken(userHomeDir + smuggoDir + apiTokenFile)
	if err != nil {
		log.Println("Error reading " + apiTokenFile + ": " + err.Error())
		log.Println("Type \"" + os.Args[0] + " apikey\" to enter your SmugMug API key.")
		os.Exit(1)
	}

	oauthClient = oauth.Client{
		TemporaryCredentialRequestURI: oauthRequestToken,
		ResourceOwnerAuthorizationURI: oauthAuthorize,
		TokenRequestURI:               oauthAccessToken,
		Credentials:                   *apiToken,
	}
}

func apikey() {
	getUserHomeDir()

	fmt.Print("Enter your SmugMug key: ")
	var key string
	if _, err := fmt.Scanln(&key); err != nil {
		fmt.Println("Reading key: " + err.Error())
		return
	}

	fmt.Print("Enter your SmugMug secret: ")
	var secret string
	if _, err := fmt.Scanln(&secret); err != nil {
		fmt.Println("Reading secret: " + err.Error())
		return
	}

	credentials := oauth.Credentials{key, secret}
	err := storeAccessData(&credentials, userHomeDir+smuggoDir+apiTokenFile)
	if err != nil {
		fmt.Println("Saving API key: " + err.Error())
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
		fmt.Println("Reading verification code: " + err.Error())
		return
	}

	accessCred, err := completeAuth(tempCred, verifyCode)
	if err != nil {
		return
	}

	fullPathTokenFile := userHomeDir + smuggoDir + userTokenFile

	if err := storeAccessData(accessCred, fullPathTokenFile); err != nil {
		log.Println("Error saving access token: " + err.Error())
		return
	}

	fmt.Println("smuggo authorized.  Access token saved to " + fullPathTokenFile)
}

// Start the auth process.
func beginAuth() (*oauth.Credentials, error) {
	tempCred, err := oauthClient.RequestTemporaryCredentials(nil, "oob", nil)
	if err != nil {
		log.Print("Error getting temp credentials: " + err.Error())
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
		log.Println("Error getting token: " + err.Error())
		return nil, err
	}
	return credentials, nil
}

// Write OAuth token or SmugMug API key to disk.
func storeAccessData(accessCred *oauth.Credentials, filename string) error {
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
