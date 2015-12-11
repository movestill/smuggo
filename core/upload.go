package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	// "net/http"
	"github.com/garyburd/go-oauth/oauth"
	"net/url"
)

const (
	apiRoot    = "https://api.smugmug.com/api/v2"
	apiCurUser = apiRoot + "!authuser"
)

func upload() {
	userToken, err := loadToken(userTokenFile)
	if err != nil {
		fmt.Println("Error reading " + userTokenFile + ": " + err.Error())
		return
	}

	userUri, err := getUser(userToken)
	if err != nil {
		return
	}

	fmt.Println(userUri)

}

// Get the URI that serves this user.
func getUser(userToken *oauth.Credentials) (string, error) {
	var queryParams = url.Values{"_accept": {"application/json"}}
	resp, err := oauthClient.Get(nil, userToken, apiCurUser, queryParams)
	if err != nil {
		fmt.Println("Error getting user endpoints: " + err.Error())
		return "", err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading user endpoints: " + err.Error())
		return "", err
	}

	var rawData interface{}
	err = json.Unmarshal(bytes, &rawData)
	if err != nil {
		fmt.Println("Error decoding user endpoints JSON: " + err.Error())
		return "", err
	}

	data := rawData.(map[string]interface{})
	rawRespObj, ok := data["Response"]
	if !ok {
		return "", errors.New("No Response object found in getUser response.")
	}

	respObj := rawRespObj.(map[string]interface{})
	rawUserObj, ok := respObj["User"]
	if !ok {
		return "", errors.New("No User object found in getUser response.")
	}
	userObj := rawUserObj.(map[string]interface{})

	rawUriObj, ok := userObj["Uri"]
	if !ok {
		return "", errors.New("No Uri object found in getUser response.")
	}

	uri := rawUriObj.(string)

	return uri, nil
}
