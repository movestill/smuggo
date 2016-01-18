#!/bin/bash
echo 'Formatting Go files.'
go fmt *.go
SOURCE_FILES="main.go auth.go upload.go albums.go"
TEST_FILES="expandFileNames_test.go aggregateTerms_test.go"

WIN32DIR="win32"
WIN64DIR="x64"
LINUX64DIR="linux64"

if [ ! -d "$WIN32DIR" ]; then
    mkdir "$WIN32DIR"
fi

if [ ! -d "$WIN64DIR" ]; then
    mkdir "$WIN64DIR"
fi

if [ ! -d "$LINUX64DIR" ]; then
    mkdir "$LINUX64DIR"
fi

case "$1" in
    test)
        echo 'Testing smuggo.'
        go test $SOURCE_FILES $TEST_FILES
        ;;
    win32)
        echo 'Building Windows x86 smuggo.exe.'
        env GOOS=windows GOARCH=386 go build -o "$WIN32DIR/smuggo.exe" $SOURCE_FILES
        ;;
    win64)
        echo 'Building Windows x64 smuggo.exe.'
        env GOOS=windows GOARCH=amd64 go build -o "$WIN64DIR/smuggo.exe" $SOURCE_FILES
        ;;
    linux64)
        echo 'Building Linux x64 smuggo.exe.'
        env GOOS=linux GOARCH=amd64 go build -o "$LINUX64DIR/smuggo" $SOURCE_FILES
        ;;
    *)
        echo 'Building OS X smuggo.'
        go build -o smuggo $SOURCE_FILES
        ;;
esac
