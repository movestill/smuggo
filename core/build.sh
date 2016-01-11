#!/bin/bash
echo 'Formatting Go files.'
go fmt *.go
SOURCE_FILES="main.go auth.go upload.go albums.go"
TEST_FILES="expandFileNames_test.go"

case "$1" in
    test)
        echo 'Testing smuggo.'
        go test $SOURCE_FILES $TEST_FILES
        ;;
    win32)
        echo 'Building Windows x86 smuggo_win32.exe.'
        env GOOS=windows GOARCH=386 go build -o smuggo_win32.exe $SOURCE_FILES
        ;;
    win64)
        echo 'Building Windows x64 smuggo_x64.exe.'
        env GOOS=windows GOARCH=amd64 go build -o smuggo_x64.exe $SOURCE_FILES
        ;;
    linux64)
        echo 'Building Linux x64 smuggo_linux64.exe.'
        env GOOS=linux GOARCH=amd64 go build -o smuggo_linux64 $SOURCE_FILES
        ;;
    *)
        echo 'Building smuggo.'
        go build -o smuggo $SOURCE_FILES
        ;;
esac
