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
    -*)
        echo 'Building smuggo.'
        go build -o smuggo $SOURCE_FILES
        ;;
esac
