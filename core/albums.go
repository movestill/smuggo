package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/go-oauth/oauth"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	apiRoot      = "https://api.smugmug.com"
	apiCurUser   = apiRoot + "/api/v2!authuser"
	apiAlbums    = "!albums"
	searchAlbums = apiRoot + "/api/v2/album!search"
)

const albumPageSize = 100

type uriJson struct {
	Uri string
}

type pagesJson struct {
	Total          int
	Start          int
	Count          int
	RequestedCount int
	NextPage       string
}

type searchAlbumJson struct {
	AlbumKey string
	UrlName  string
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
	Album             []albumJson
	Pages             pagesJson
	User              uriJson
	AlbumSearchResult []searchAlbumJson
}

type responseJson struct {
	Response endpointJson
}

// getUser retrieves the URI that serves the current user.
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

	if resp.StatusCode != http.StatusOK {
		fmt.Println("getUser response: " + resp.Status)
	}

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

// printAlbums prints all the albums after sorting alphabetically.
func printAlbums(albums []albumJson) {
	sort.Sort(byUrlName(albums))
	for _, album := range albums {
		tokens := strings.Split(album.Uri, "/")
		if len(tokens) > 0 {
			fmt.Println(album.UrlName + " :: " + tokens[len(tokens)-1])
		}
	}
}

// aggregateTerms combines search terms into a single string with each search
// term separated by a plus sign.
func aggregateTerms(terms []string) string {
	var combinedTerms string
	for i, term := range terms {
		combinedTerms += term
		if i < len(terms)-1 {
			combinedTerms += "+"
		}
	}

	return combinedTerms
}

// search is the entry point to album search.
func search(terms []string) {
	userToken, err := loadUserToken()
	if err != nil {
		fmt.Println("Error reading OAuth token: " + err.Error())
		return
	}

	userUri, err := getUser(userToken)
	if err != nil {
		return
	}

	combinedTerms := aggregateTerms(terms)
	var client = http.Client{}

	searchRequest(&client, userToken, userUri, combinedTerms, 1)
}

// searchRequest sends the search request to SmugMug and asks for the entries beginning at start.
func searchRequest(client *http.Client, userToken *oauth.Credentials, userUri string, query string, start int) {
	var queryParams = url.Values{
		"_accept":       {"application/json"},
		"_verbosity":    {"1"},
		"Scope":         {userUri},
		"SortDirection": {"Descending"},
		"SortMethod":    {"Rank"},
		"Text":          {query},
		"start":         {fmt.Sprintf("%d", start)},
		"count":         {"15"},
	}

	resp, err := oauthClient.Get(client, userToken, searchAlbums, queryParams)
	if err != nil {
		return
	}

	bytes, err := func() ([]byte, error) {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return b, nil
	}()

	if err != nil {
		fmt.Println("Reading search results: " + err.Error())
		return
	}

	var respJson responseJson
	err = json.Unmarshal(bytes, &respJson)
	if err != nil {
		fmt.Println("Decoding album search endpoint JSON: " + err.Error())
		return
	}

	if len(respJson.Response.AlbumSearchResult) < 1 {
		fmt.Println("No search results found.")
		return
	}

	printSearchResults(respJson.Response.AlbumSearchResult)

	pages := &respJson.Response.Pages
	if pages.Count+pages.Start < pages.Total {
		fmt.Println("Press Enter for more results or Ctrl-C to quit.")
		var foo string
		fmt.Scanln(&foo)
		searchRequest(client, userToken, userUri, query, pages.Count+pages.Start)
	}
}

// printSearchResults outputs the album names and keys to stdout.
func printSearchResults(results []searchAlbumJson) {
	for _, album := range results {
		fmt.Println(album.UrlName + " :: " + album.AlbumKey)
	}
}

// albums lists all the albums (and their keys) that belong to the user.
func albums() {
	userToken, err := loadUserToken()
	if err != nil {
		fmt.Println("Error reading OAuth token: " + err.Error())
		return
	}

	userUri, err := getUser(userToken)
	if err != nil {
		return
	}

	startT := time.Now()
	albumsUri := apiRoot + userUri + apiAlbums
	var client = http.Client{}
	epChan := make(chan endpointJson, 10)
	fmt.Println("Requesting number of albums.")
	getAlbumPage(&client, userToken, albumsUri, 1, 1, epChan)
	ep := <-epChan

	if ep.Pages.Count >= ep.Pages.Total {
		printAlbums(ep.Album)
		return
	}

	waitGrp := sync.WaitGroup{}
	start := ep.Pages.Count + 1

	for start < ep.Pages.Total {
		fmt.Printf("Requesting %d albums starting at %d.\n", albumPageSize, start)
		waitGrp.Add(1)
		go func(startInd int) {
			defer waitGrp.Done()
			getAlbumPage(&client, userToken, albumsUri, startInd, albumPageSize, epChan)
		}(start)
		start += albumPageSize
	}

	albums := make([]albumJson, 0, ep.Pages.Total)
	copy(albums, ep.Album)

	albumsReqDoneChan := make(chan bool)
	resultsPrintedChan := make(chan bool)
	go collectAlbumResults(albums, albumsReqDoneChan, epChan,
		resultsPrintedChan)

	waitGrp.Wait()

	// Tell collectAlbumResults() that all album requests finished.
	albumsReqDoneChan <- true

	// Wait for albums to be displayed.
	<-resultsPrintedChan
	totalT := time.Since(startT)
	fmt.Println("\nElapsed time: " + totalT.String())
}

// collectAlbumResults receives albums over epChan from getAlbumPage().  It
// continues to listen to epChan until receiving true from albumsReqDoneChan.
// Finally, it outputs the albums to stdout and indicates completion by
// sending true over resultsPrintedChan.
func collectAlbumResults(
	albums []albumJson,
	albumsReqDoneChan chan bool,
	epChan chan endpointJson,
	resultsPrintedChan chan bool) {

	done := false
	for !done || len(epChan) > 0 {
		select {
		case epAlbs := <-epChan:
			albums = append(albums, epAlbs.Album...)
		case done = <-albumsReqDoneChan:
		}
	}

	printAlbums(albums)
	resultsPrintedChan <- true
}

// getAlbumPage gets up to count albums starting at index start.  It returns
// the album and page data over epChan, so it may be invoked as a goroutine.
func getAlbumPage(
	client *http.Client, userToken *oauth.Credentials,
	albumsUri string, start int, count int,
	epChan chan endpointJson) {

	var queryParams = url.Values{
		"_accept":    {"application/json"},
		"_verbosity": {"1"},
		"start":      {fmt.Sprintf("%d", start)},
		"count":      {fmt.Sprintf("%d", count)},
	}

	resp, err := oauthClient.Get(client, userToken, albumsUri, queryParams)
	if err != nil {
		return
	}

	bytes, err := func() ([]byte, error) {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return b, nil
	}()

	if err != nil {
		fmt.Println("Reading albums: " + err.Error())
		return
	}

	var respJson responseJson
	err = json.Unmarshal(bytes, &respJson)
	if err != nil {
		fmt.Println("Decoding album endpoint JSON: " + err.Error())
		return
	}

	if len(respJson.Response.Album) < 1 {
		fmt.Println("No albums found.")
		return
	}

	epChan <- respJson.Response
}

// createAlbum was test code for exercising the SmugMug API.  It works, but is
// hard coded for a particular album in a particular location.
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
