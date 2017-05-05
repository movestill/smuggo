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
	//"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	//"path/filepath"
	//"reflect"
	"testing"
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
func (c *CountHandler) PassResponse(resp http.ResponseWriter, req *http.Request) {
	c.numRequests++
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("{\"stat\": \"pass\"}"))
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

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Log(err.Error())
		os.Exit(1)
	}
	f.Close()
	nTries := uint(3)

	err = postImage(&client, server.URL, userToken, albumKey, filename, nTries)

	if handler.numRequests != nTries {
		t.Errorf("Expected %d tries, actual %d", nTries, handler.numRequests)
	}

	err = os.Remove(filename)
}

// Test no retries when first upload attempt succeeds.
func TestUploadSuccess(t *testing.T) {
	handler := CountHandler{}
	server := httptest.NewServer(http.HandlerFunc(handler.PassResponse))
	defer server.Close()

	getUserHomeDir()
	userToken, err := loadUserToken()
	if err != nil {
		t.Log("Error reading OAuth token: " + err.Error())
		return
	}

	var client = http.Client{}

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Log(err.Error())
		os.Exit(1)
	}
	f.Close()
	nTries := uint(3)

	err = postImage(&client, server.URL, userToken, albumKey, filename, nTries)
	if err != nil {
		t.Log("Error uploading: " + err.Error())
	}

	expTries := uint(1)
	if handler.numRequests != expTries {
		t.Errorf("Expected %d tries, actual %d", expTries, handler.numRequests)
	}

	err = os.Remove(filename)
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

	albumKey := "foo"
	filename := "fake_image.png"
	f, err := os.Create(filename)
	if err != nil {
		t.Log(err.Error())
		os.Exit(1)
	}
	f.Close()
	nTries := uint(3)

	err = postImage(&client, server.URL, userToken, albumKey, filename, nTries)

	if handler.numRequests != nTries {
		t.Errorf("Expected %d tries, actual %d", nTries, handler.numRequests)
	}

	err = os.Remove(filename)
}
