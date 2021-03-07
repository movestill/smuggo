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
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gomodule/oauth1/oauth"
)

const uploadURI = "https://upload.smugmug.com/"

type uploadResponseJSON struct {
	Stat    string
	Message string
}

// Returns true if image's MD5 hash already exists in the given album.  If there
// are duplicates, an array of filenames of the duplicates is also returned.  If
// an error reading the DB occurs, will return false under the assumption that
// it's better to allow a duplicate upload than to abort.
func isDuplicateImage(db *sql.DB, albumKey string, hash string) (bool, []string) {
	filenames := getDuplicateImages(db, albumKey, hash)
	dupe := len(filenames) > 0

	return dupe, filenames
}

// upload transfers a single file to the SmugMug album identifed by key.
func upload(allowDupes bool, albumKey string, filename string) {
	userToken, err := loadUserToken()
	if err != nil {
		log.Println("Error reading OAuth token: " + err.Error())
		return
	}

	var client = http.Client{}
	db := openDB()
	defer db.Close()

	err = postImage(&client, uploadURI, userToken, db, allowDupes, albumKey, filename, retriesFlag+1)
	if err != nil {
		log.Println("Error uploading: " + err.Error())
	}
}

// expandFileNames applies pattern matching to the given list of filenames.
// Pass filepath.Glob as the expander function.  The pattern matching function
// is a parameter for testing purposes.
func expandFileNames(
	filenames []string, expander func(pattern string) ([]string, error)) []string {

	expanded := make([]string, 0, 20)

	for _, fname := range filenames {
		matches, err := expander(fname)
		if err != nil {
			continue
		}
		expanded = append(expanded, matches...)
	}

	return expanded
}

// multiUpload uploads files in parallel to the given SmugMug album.
func multiUpload(numParallel int, allowDupes bool, albumKey string, filenames []string) {
	if numParallel < 1 {
		log.Println("Error, must upload at least 1 file at a time!")
		return
	}

	userToken, err := loadUserToken()
	if err != nil {
		log.Println("Error reading OAuth token: " + err.Error())
		return
	}

	expFileNames := expandFileNames(filenames, filepath.Glob)
	fmt.Println(expFileNames)
	var client = http.Client{}
	db := openDB()
	defer db.Close()

	semaph := make(chan int, numParallel)
	for _, filename := range expFileNames {
		semaph <- 1
		go func(filename string) {
			fmt.Println("go " + filename)
			err := postImage(&client, uploadURI, userToken, db, allowDupes, albumKey, filename, retriesFlag+1)
			if err != nil {
				log.Println("Error uploading: " + err.Error())
			}
			<-semaph
		}(filename)
	}

	for {
		time.Sleep(time.Second)
		if len(semaph) == 0 {
			break
		}
	}
}

// getMediaType determines the value for the Content-Type header field based
// on the file extension.
func getMediaType(filename string) string {
	ext := filepath.Ext(filename)
	return mime.TypeByExtension(ext)
}

// calcMD5 generates the MD5 sum for the given file.
func calcMD5(imgFileName string) (string, int64, error) {
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

// postImage uploads a single image to SmugMug via the POST method.
// uri is the protocol + hostname of the server
func postImage(client *http.Client, uri string, credentials *oauth.Credentials,
	db *sql.DB, allowDupes bool,
	albumKey string, imgFileName string, tries uint) error {

	md5Str, imgSize, err := calcMD5(imgFileName)
	if err != nil {
		return err
	}

	if !allowDupes {
		isDupe, filenames := isDuplicateImage(db, albumKey, md5Str)
		if isDupe {
			fmt.Printf("Not uploading %s, duplicate images in album:\n", imgFileName)
			for _, f := range filenames {
				fmt.Printf("\t%s\n", f)
			}
			return nil
		}
	}

	var success = false
	var tryCount uint
	for tryCount = 0; tryCount < tries; tryCount++ {

		file, err := os.Open(imgFileName)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", uri, file)
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
			"Content-Type":        {getMediaType(justImgFileName)},
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
			return err
		}

		var resp *http.Response
		resp, err = client.Do(req)
		if err != nil {
			log.Println("Error sending POST request: " + err.Error())
			if tryCount < tries-1 {
				continue
			}
			return err
		}

		defer resp.Body.Close()

		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response: " + err.Error())
			if tryCount < tries-1 {
				continue
			}
			return err
		}

		fmt.Println(resp.Status)
		fmt.Println(string(bytes))

		var respJSON uploadResponseJSON
		err = json.Unmarshal(bytes, &respJSON)
		if err != nil {
			log.Println("Error decoding upload response JSON: " + err.Error())
			if tryCount < tries-1 {
				continue
			}
			return err
		}

		if respJSON.Stat == "ok" {
			success = true
			break
		}
	}

	if success {
		imgData := []imageJSON{{md5Str, imgFileName}}
		writeImageData(db, albumKey, imgData)
		return nil
	}

	return fmt.Errorf("SmugMug unable to receive image after %d attempts", tries)
}
