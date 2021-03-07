// Copyright 2016 Timothy Gion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/gomodule/oauth1/oauth"
)

const (
	apiRoot        = "https://api.smugmug.com"
	apiCurUser     = apiRoot + "/api/v2!authuser"
	apiMultiAlbums = "!albums"
	apiAlbum       = apiRoot + "/api/v2/album"
	searchAlbums   = apiAlbum + "!search"
)

const albumPageSize = 100

type uriJSON struct {
	URI string
}

type pagesJSON struct {
	Total          int // Total number of albums.
	Start          int // Index of first album for the current page (starts at 1).
	Count          int // Number of albums returned for current page.
	RequestedCount int // Requested number of albums.
}

type albumJSON struct {
	AlbumKey string
	Name     string
}

// Sort album array by Name for printing.
type byName []albumJSON

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return b[i].Name < b[j].Name }

type endpointJSON struct {
	Album []albumJSON
	Pages pagesJSON
	User  uriJSON
}

// Standard top level response from SmugMug API.
type responseJSON struct {
	Response endpointJSON
}

type searchJSON struct {
	Album []albumJSON
	Pages pagesJSON
}

// Top level response for search from SmugMug API.
type searchResponseJSON struct {
	Response searchJSON
}

type imageJSON struct {
	ArchivedMD5 string
	FileName    string
}

type imagesPagesJSON struct {
	Total          int
	Start          int
	Count          int
	RequestedCount int
}

type imagesJSON struct {
	AlbumImage []imageJSON
	Pages      pagesJSON
}

// Top level response from the AlbumImages URI.  We just want the MD5 for all
// the images so we can avoid uploading duplicates.
type imagesResponseJSON struct {
	Response imagesJSON
}

// getUser retrieves the URI that serves the current user.
func getUser(userToken *oauth.Credentials) (string, error) {
	var queryParams = url.Values{
		"_accept":    {"application/json"},
		"_verbosity": {"1"},
	}
	resp, err := oauthClient.Get(nil, userToken, apiCurUser, queryParams)
	if err != nil {
		log.Println("Error getting user endpoint: " + err.Error())
		return "", err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading user endpoint: " + err.Error())
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println("getUser response: " + resp.Status)
	}

	var respJSON responseJSON
	err = json.Unmarshal(bytes, &respJSON)
	if err != nil {
		log.Println("Error decoding user endpoint JSON: " + err.Error())
		return "", err
	}

	if respJSON.Response.User.URI == "" {
		fmt.Println("No Uri object found in getUser response.")
		return "", errors.New("no Uri object found in getUser response")
	}

	return respJSON.Response.User.URI, nil
}

// printAlbums prints all the albums after sorting alphabetically.
func printAlbums(albums []albumJSON) {
	sort.Sort(byName(albums))
	for _, album := range albums {
		fmt.Println(album.Name + " :: " + album.AlbumKey)
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
		log.Println("Error reading OAuth token: " + err.Error())
		return
	}

	userURI, err := getUser(userToken)
	if err != nil {
		return
	}

	combinedTerms := aggregateTerms(terms)
	var client = http.Client{}

	searchRequest(&client, userToken, userURI, combinedTerms, 1)
}

// searchRequest sends the search request to SmugMug and asks for the entries beginning at start.
func searchRequest(client *http.Client, userToken *oauth.Credentials, userURI string, query string, start int) {
	var queryParams = url.Values{
		"_accept":       {"application/json"},
		"_verbosity":    {"1"},
		"_filter":       {"Album,Name,AlbumKey"},
		"_filteruri":    {""},
		"Scope":         {userURI},
		"SortDirection": {"Descending"},
		"SortMethod":    {"Rank"},
		"Text":          {query},
		"start":         {fmt.Sprintf("%d", start)},
		"count":         {"15"},
	}

	resp, err := oauthClient.Get(client, userToken, searchAlbums, queryParams)
	if err != nil {
		fmt.Println(err.Error())
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
		log.Println("Reading search results: " + err.Error())
		return
	}

	var respJSON searchResponseJSON
	err = json.Unmarshal(bytes, &respJSON)
	if err != nil {
		log.Println("Decoding album search endpoint JSON: " + err.Error())
		return
	}

	if len(respJSON.Response.Album) < 1 {
		fmt.Println("No search results found.")
		return
	}

	printSearchResults(respJSON.Response.Album)

	pages := &respJSON.Response.Pages
	if pages.Count+pages.Start < pages.Total {
		fmt.Println("Press Enter for more results or Ctrl-C to quit.")
		var foo string
		fmt.Scanln(&foo)
		searchRequest(client, userToken, userURI, query, pages.Count+pages.Start)
	}
}

// printSearchResults outputs the album names and keys to stdout.
func printSearchResults(results []albumJSON) {
	for _, album := range results {
		fmt.Println(album.Name + " :: " + album.AlbumKey)
	}
}

// albums lists all the albums (and their keys) that belong to the user.
func albums() {
	userToken, err := loadUserToken()
	if err != nil {
		log.Println("Error reading OAuth token: " + err.Error())
		return
	}

	userURI, err := getUser(userToken)
	if err != nil {
		return
	}

	startT := time.Now()
	albumsURI := apiRoot + userURI + apiMultiAlbums
	var client = http.Client{}
	epChan := make(chan endpointJSON, 10)
	fmt.Println("Requesting number of albums.")
	getAlbumPage(&client, userToken, albumsURI, 1, 1, epChan)
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
			getAlbumPage(&client, userToken, albumsURI, startInd, albumPageSize, epChan)
		}(start)
		start += albumPageSize
	}

	albums := make([]albumJSON, 1, ep.Pages.Total)
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
	albums []albumJSON,
	albumsReqDoneChan chan bool,
	epChan chan endpointJSON,
	resultsPrintedChan chan bool) {

	fmt.Println(albums)
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
	albumsURI string, start int, count int,
	epChan chan endpointJSON) {

	var queryParams = url.Values{
		"_accept":    {"application/json"},
		"_verbosity": {"1"},
		"_filter":    {"AlbumKey,Name"},
		"_filteruri": {""},
		"start":      {fmt.Sprintf("%d", start)},
		"count":      {fmt.Sprintf("%d", count)},
	}

	resp, err := oauthClient.Get(client, userToken, albumsURI, queryParams)
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
		log.Println("Reading albums: " + err.Error())
		return
	}

	var respJSON responseJSON
	err = json.Unmarshal(bytes, &respJSON)
	if err != nil {
		log.Println("Decoding album endpoint JSON: " + err.Error())
		return
	}

	if len(respJSON.Response.Album) < 1 {
		fmt.Println("No albums found.")
		return
	}

	epChan <- respJSON.Response
}

// createAlbum was test code for exercising the SmugMug API.  It works, but is
// hard coded for a particular album in a particular location.
func createAlbum(client *http.Client, credentials *oauth.Credentials) {
	createURI := apiRoot + "/api/v2/node/R3gfM!children"

	var body = map[string]string{
		"Type":    "Album",
		"Name":    "Test Post Create",
		"UrlName": "Test-Post-Create",
		"Privacy": "Public",
	}

	rawJSON, err := json.Marshal(body)
	if err != nil {
		return
	}
	fmt.Println(string(rawJSON))

	req, err := http.NewRequest("POST", createURI, bytes.NewReader(rawJSON))
	if err != nil {
		return
	}

	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["Content-Length"] = []string{fmt.Sprintf("%d", len(rawJSON))}
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

// albumImages retrieves the MD5 hash codes of the images in an album.
func albumImages(albumKey string) {

	userToken, err := loadUserToken()
	if err != nil {
		log.Println("Error reading OAuth token: " + err.Error())
		return
	}

	uri := apiAlbum + "/" + albumKey + "!images"
	var client = http.Client{}
	imgsChan := make(chan imagesJSON, 10)
	getAlbumImagesPage(&client, userToken, uri, 1, 1, imgsChan)
	img := <-imgsChan

	fmt.Printf("Got %d images out of %d.\n", img.Pages.Count, img.Pages.Total)
	if img.Pages.Count >= img.Pages.Total {
		db := openDB()
		defer db.Close()

		// First remove any image data for the given album because we are getting
		// new truth data.
		removeAlbumImages(db, albumKey)
		writeImageData(db, albumKey, img.AlbumImage)
		return
	}

	waitGrp := sync.WaitGroup{}
	start := img.Pages.Count + 1

	for start < img.Pages.Total {
		fmt.Printf("Requesting %d images starting at %d.\n", albumPageSize, start)
		waitGrp.Add(1)
		go func(startInd int) {
			defer waitGrp.Done()
			getAlbumImagesPage(&client, userToken, uri, startInd, albumPageSize, imgsChan)
		}(start)
		start += albumPageSize
	}

	imgData := make([]imageJSON, img.Pages.Count, img.Pages.Total)
	copy(imgData, img.AlbumImage)

	imgsReqDoneChan := make(chan bool)
	gotAllImgsChan := make(chan bool)
	go collectImageResults(imgData, albumKey, imgsReqDoneChan, imgsChan,
		gotAllImgsChan)

	waitGrp.Wait()

	// Tell collectImageResults() that all album image requests finished.
	imgsReqDoneChan <- true

	// Wait for image data collection to complete.
	<-gotAllImgsChan
}

// collectImageResults receives albums over imgsChan from getAlbumImagesPage().
// It continues to listen to imgsChan until receiving true from imgsReqDoneChan.
// Finally, it indicates completion by sending true over gotAllImgsChan.
func collectImageResults(imgData []imageJSON, albumKey string,
	imgsReqDoneChan chan bool, imgsChan chan imagesJSON, gotAllImgsChan chan bool) {

	done := false
	for !done || len(imgsChan) > 0 {
		select {
		case newImages := <-imgsChan:
			imgData = append(imgData, newImages.AlbumImage...)
		case done = <-imgsReqDoneChan:
		}
	}

	fmt.Println("Got", len(imgData), "images")

	db := openDB()
	defer db.Close()

	// First remove any image data for the given album because we are getting
	// new truth data.
	removeAlbumImages(db, albumKey)

	// Save the data received from SmugMug.
	writeImageData(db, albumKey, imgData)

	gotAllImgsChan <- true
}

func getAlbumImagesPage(
	client *http.Client, userToken *oauth.Credentials,
	uri string, start int, count int,
	imgsChan chan imagesJSON) {

	var queryParams = url.Values{
		"_accept":    {"application/json"},
		"_verbosity": {"1"},
		"_filter":    {"ArchivedMD5,FileName"},
		"_filteruri": {""},
		"start":      {fmt.Sprintf("%d", start)},
		"count":      {"count"},
	}

	resp, err := oauthClient.Get(client, userToken, uri, queryParams)
	if err != nil {
		fmt.Println(err.Error())
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
		log.Println("Reading album images: " + err.Error())
		return
	}

	var respJSON imagesResponseJSON
	err = json.Unmarshal(bytes, &respJSON)
	if err != nil {
		log.Println("Decoding album images endpoint JSON: " + err.Error())
		return
	}

	imgsChan <- respJSON.Response
}
