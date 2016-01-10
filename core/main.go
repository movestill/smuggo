package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/go-oauth/oauth"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// The names of the token files.
const (
	apiTokenFile  = "apiToken.json"
	userTokenFile = "userToken.json"
)

// loadToken imports tokens from the given JSON file.
func loadToken(filename string) (*oauth.Credentials, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var token oauth.Credentials
	if err := json.Unmarshal(bytes, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// usage gives minimal usage instructions.
func usage() {
	fmt.Println("Usage: ")
	fmt.Println(os.Args[0] + " apikey|auth|albums|upload|multiupload")
	fmt.Println("\tapikey")
	fmt.Println("\tauth")
	fmt.Println("\talbums")
	fmt.Println("\tupload <album key> <filename>")
	fmt.Println("\tmultiupload <# parallel uploads> <album key> <filename 1> ... <filename n>")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	loweredCmd := strings.ToLower(os.Args[1])
	if loweredCmd == "apikey" {
		apikey()
		return
	}

	// Normal code path where an API key must exist.
	authInit()

	switch loweredCmd {
	case "auth":
		auth()
	case "upload":
		if len(os.Args) != 4 {
			usage()
			return
		}
		upload(os.Args[2], os.Args[3])
	case "albums":
		albums()
	case "multiupload":
		if len(os.Args) < 5 {
			usage()
			return
		}
		numParallel, err := strconv.Atoi(os.Args[2])
		if err != nil {
			usage()
			return
		}
		multiUpload(numParallel, os.Args[3], os.Args[4:])
	default:
		usage()
		return
	}
}
