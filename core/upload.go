package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/go-oauth/oauth"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

const (
	apiRoot    = "https://api.smugmug.com"
	apiCurUser = apiRoot + "/api/v2!authuser"
	apiAlbums  = "!albums"
)

func upload(albumKey string, filename string) {
	userToken, err := loadToken(userTokenFile)
	if err != nil {
		fmt.Println("Error reading " + userTokenFile + ": " + err.Error())
		return
	}

	userUri, err := getUser(userToken)
	if err != nil {
		return
	}

	var client = http.Client{}
	// createAlbum(&client, userToken)

	err = postImage(&client, userToken, albumKey, filename)
	if err != nil {
		fmt.Println("Error uploading: " + err.Error())
	}
}

func createAlbum(client *http.Client, credentials *oauth.Credentials) {
	createUri := apiRoot + "/api/v2/node/R3gfM!children"

	var body = map[string]string{
		"Type":    "Album",
		"Name":    "Test Post Create",
		"UrlName": "Test-Post-Create",
		"Privacy": "Public",
	}

	rawJson, err := json.Marshal(body)
	if err != nil {
		return
	}
	fmt.Println(string(rawJson))

	req, err := http.NewRequest("POST", createUri, bytes.NewReader(rawJson))
	if err != nil {
		return
	}

	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["Content-Length"] = []string{fmt.Sprintf("%d", len(rawJson))}
	req.Header["Accept"] = []string{"application/json"}

	if err := oauthClient.SetAuthorizationHeader(
		req.Header, credentials, "POST", req.URL, url.Values{}); err != nil {
		// req.Header, credentials, "POST", req.URL, headers); err != nil {
		return
	}

	fmt.Println(req)

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	fmt.Println(resp.Status)
	fmt.Println(string(bytes))
}

func calcMd5(imgFileName string) (string, int64, error) {
	file, err := os.Open(imgFileName)
	if err != nil {
		return "", 0, err
	}

	defer file.Close()

	hash := md5.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}

	var md5Sum []byte
	md5Sum = hash.Sum(md5Sum)
	return fmt.Sprintf("%x", md5Sum), size, nil
}

func postImage(client *http.Client, credentials *oauth.Credentials,
	albumKey string, imgFileName string) error {

	md5Str, imgSize, err := calcMd5(imgFileName)
	if err != nil {
		return err
	}

	uploadUri := "https://upload.smugmug.com/"
	file, err := os.Open(imgFileName)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", uploadUri, file)
	if err != nil {
		return err
	}

	req.ContentLength = imgSize

	for key, val := range oauthClient.Header {
		req.Header[key] = val
	}

	_, justImgFileName := filepath.Split(imgFileName)
	var headers = url.Values{
		"Accept":              {"application/json"},
		"Content-Type":        {"image/jpeg"},
		"Content-MD5":         {md5Str},
		"Content-Length":      {strconv.FormatInt(imgSize, 10)},
		"X-Smug-ResponseType": {"JSON"},
		"X-Smug-AlbumUri":     {"/api/v2/album/" + albumKey},
		"X-Smug-Version":      {"v2"},
		"X-Smug-Filename":     {justImgFileName},
	}

	for key, val := range headers {
		req.Header[key] = val
	}

	if err := oauthClient.SetAuthorizationHeader(
		req.Header, credentials, "POST", req.URL, url.Values{}); err != nil {
		return nil
	}

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(resp.Status)
	fmt.Println(string(bytes))

	return nil
}

type uriJson struct {
	Uri string
}

type albumJson struct {
	Uri     string
	UrlName string
}

// Sort album array by UrlName for printing.
type byUrlName []albumJson

func (b byUrlName) Len() int           { return len(b) }
func (b byUrlName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byUrlName) Less(i, j int) bool { return b[i].UrlName < b[j].UrlName }

type endpointJson struct {
	Album []albumJson
	User  uriJson
}

type responseJson struct {
	Response endpointJson
}

// Get the URI that serves this user.
func getUser(userToken *oauth.Credentials) (string, error) {
	var queryParams = url.Values{
		"_accept":    {"application/json"},
		"_verbosity": {"1"},
	}
	resp, err := oauthClient.Get(nil, userToken, apiCurUser, queryParams)
	if err != nil {
		fmt.Println("Error getting user endpoint: " + err.Error())
		return "", err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading user endpoint: " + err.Error())
		return "", err
	}

	fmt.Println("getUser response: " + resp.Status)

	var respJson responseJson
	err = json.Unmarshal(bytes, &respJson)
	if err != nil {
		fmt.Println("Error decoding user endpoint JSON: " + err.Error())
		return "", err
	}

	if respJson.Response.User.Uri == "" {
		fmt.Println("No Uri object found in getUser response.")
		return "", errors.New("No Uri object found in getUser response.")
	}

	return respJson.Response.User.Uri, nil
}

func albums() {
	userToken, err := loadToken(userTokenFile)
	if err != nil {
		fmt.Println("Error reading " + userTokenFile + ": " + err.Error())
		return
	}

	userUri, err := getUser(userToken)
	if err != nil {
		return
	}

	albumsUri := apiRoot + userUri + apiAlbums
	fmt.Println(albumsUri)

	var queryParams = url.Values{
		"_accept": {"application/json"},
		// "filter":     {"Album"},
		// "filteruri":  {""},
		"_verbosity": {"1"},
	}
	resp, err := oauthClient.Get(nil, userToken, albumsUri, queryParams)
	if err != nil {
		fmt.Println("Error getting user endpoint: " + err.Error())
		return
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading album endpoint: " + err.Error())
		return
	}

	fmt.Println(string(bytes))

	var respJson responseJson
	err = json.Unmarshal(bytes, &respJson)
	if err != nil {
		fmt.Println("Error decoding album endpoint JSON: " + err.Error())
		return
	}

	if len(respJson.Response.Album) < 1 {
		fmt.Println("No albums found.")
		return
	}

	sort.Sort(byUrlName(respJson.Response.Album))

	for _, album := range respJson.Response.Album {
		fmt.Println(album.UrlName + " :: " + album.Uri)
	}
}
