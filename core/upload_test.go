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
	"database/sql"
	"fmt"

	//"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	//"path/filepath"
	//"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type HangupHandler struct {
	server      *httptest.Server
	numRequests uint
}

// Count request and rudely hangup connection.
func (h *HangupHandler) DisconnectResponse(resp http.ResponseWriter, req *http.Request) {
	h.numRequests++
	h.server.CloseClientConnections()
}

type CountHandler struct {
	numRequests uint
}

// Count request and indicate failure.
func (c *CountHandler) FailResponse(resp http.ResponseWriter, req *http.Request) {
	c.numRequests++
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("{\"stat\": \"fail\"}"))
}

// Count request and indicate success.
func (c *CountHandler) OkResponse(resp http.ResponseWriter, req *http.Request) {
	c.numRequests++
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("{\"stat\": \"ok\"}"))
}

// Test that there are 3 tries when the server breaks the connection.
func TestServerHangsUp(t *testing.T) {
	handler := HangupHandler{}
	server := httptest.NewServer(http.HandlerFunc(handler.DisconnectResponse))
	handler.server = server
	defer server.Close()

	getUserHomeDir()
	userToken, err := loadUserToken()
	if err != nil {
		t.Log("Error reading OAuth token: " + err.Error())
		return
	}

	var client = http.Client{}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	defer db.Close()

	createTables(db, 0)

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Log(err.Error())
		os.Exit(1)
	}
	f.Close()
	defer os.Remove(filename)
	allowDupes := true
	nTries := uint(3)

	err = postImage(&client, server.URL, userToken, db, allowDupes, albumKey, filename, nTries)
	if err == nil {
		t.Error("Expected error from postImage()")
	}

	if handler.numRequests != nTries {
		t.Errorf("Expected %d tries, actual %d", nTries, handler.numRequests)
	}
}

// Test no retries when first upload attempt succeeds.
func TestUploadSuccess(t *testing.T) {
	handler := CountHandler{}
	server := httptest.NewServer(http.HandlerFunc(handler.OkResponse))
	defer server.Close()

	getUserHomeDir()
	userToken, err := loadUserToken()
	if err != nil {
		t.Log("Error reading OAuth token: " + err.Error())
		return
	}

	var client = http.Client{}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	defer db.Close()

	createTables(db, 0)

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Error("Error creating fake image", err)
	}
	f.Close()
	defer os.Remove(filename)
	allowDupes := true
	nTries := uint(3)

	err = postImage(&client, server.URL, userToken, db, allowDupes, albumKey, filename, nTries)
	if err != nil {
		t.Error("Error uploading: ", err)
	}

	expTries := uint(1)
	if handler.numRequests != expTries {
		t.Errorf("Expected %d tries, actual %d", expTries, handler.numRequests)
	}
}

func TestUploadSuccessWritesImageDataToDB(t *testing.T) {
	handler := CountHandler{}
	server := httptest.NewServer(http.HandlerFunc(handler.OkResponse))
	defer server.Close()

	getUserHomeDir()
	userToken, err := loadUserToken()
	if err != nil {
		t.Log("Error reading OAuth token: " + err.Error())
		return
	}

	var client = http.Client{}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	defer db.Close()

	createTables(db, 0)

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Error("Error creating fake image", err)
	}
	f.Close()
	defer os.Remove(filename)
	allowDupes := true
	nTries := uint(3)

	err = postImage(&client, server.URL, userToken, db, allowDupes, albumKey, filename, nTries)
	if err != nil {
		t.Error("Error uploading: ", err)
	}

	rows, err := db.Query(fmt.Sprintf("SELECT album_key, filename FROM %s;", imageTable))
	if err != nil {
		t.Error(err)
	}

	var count = 0
	for rows.Next() {
		count++
		var actualAlbumKey string
		var actualFilename string
		err = rows.Scan(&actualAlbumKey, &actualFilename)
		if err != nil {
			t.Error(err)
		}
		if actualAlbumKey != albumKey {
			t.Errorf("Expected album key %s, got %s\n", albumKey, actualAlbumKey)
		}
		if actualFilename != filename {
			t.Errorf("Expected filename %s, got %s\n", actualFilename, filename)
		}
	}
	if count != 1 {
		t.Errorf("Expected 1 row in image table but found %d", count)
	}
}

// Test trying 3 times when server indicates upload failure.
func TestUploadRetries(t *testing.T) {
	handler := CountHandler{}
	server := httptest.NewServer(http.HandlerFunc(handler.FailResponse))
	defer server.Close()

	getUserHomeDir()
	userToken, err := loadUserToken()
	if err != nil {
		t.Log("Error reading OAuth token: " + err.Error())
		return
	}

	var client = http.Client{}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	defer db.Close()

	createTables(db, 0)

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Error("Error creating fake image", err)
	}
	f.Close()
	defer os.Remove(filename)
	allowDupes := true
	nTries := uint(3)

	err = postImage(&client, server.URL, userToken, db, allowDupes, albumKey, filename, nTries)
	if err == nil {
		t.Error("Expected error from postImage()")
	}

	if handler.numRequests != nTries {
		t.Errorf("Expected %d tries, actual %d", nTries, handler.numRequests)
	}
}

func TestUploadFailureDoesNotWriteImageDataToDB(t *testing.T) {
	handler := CountHandler{}
	server := httptest.NewServer(http.HandlerFunc(handler.FailResponse))
	defer server.Close()

	getUserHomeDir()
	userToken, err := loadUserToken()
	if err != nil {
		t.Log("Error reading OAuth token: " + err.Error())
		return
	}

	var client = http.Client{}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	defer db.Close()

	createTables(db, 0)

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Error("Error creating fake image", err)
	}
	f.Close()
	defer os.Remove(filename)
	allowDupes := true
	nTries := uint(1)

	err = postImage(&client, server.URL, userToken, db, allowDupes, albumKey, filename, nTries)
	if err == nil {
		t.Error("Expected error from postImage()")
	}

	row := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s;", imageTable))
	var count uint
	err = row.Scan(&count)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Errorf("Expected 0 rows in image table but found %d", count)
	}
}

func TestDoesNotUploadDuplicateImage(t *testing.T) {
	handler := CountHandler{}
	server := httptest.NewServer(http.HandlerFunc(handler.OkResponse))
	defer server.Close()

	getUserHomeDir()
	userToken, err := loadUserToken()
	if err != nil {
		t.Log("Error reading OAuth token: " + err.Error())
		return
	}

	var client = http.Client{}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	defer db.Close()

	createTables(db, 0)

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Error("Error creating fake image", err)
	}
	f.Close()
	defer os.Remove(filename)

	hash, _, err := calcMD5(filename)
	imgData := []imageJSON{{hash, filename}}
	writeImageData(db, albumKey, imgData)

	allowDupes := false
	nTries := uint(1)

	err = postImage(&client, server.URL, userToken, db, allowDupes, albumKey, filename, nTries)

	if handler.numRequests > 0 {
		t.Error("Failed to detect duplicate image; should not have tried to upload")
	}

	if err != nil {
		t.Error(err)
	}
}
