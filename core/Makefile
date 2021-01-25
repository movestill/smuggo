SOURCE = main.go auth.go upload.go albums.go
TEST = expandFileNames_test.go aggregateTerms_test.go upload_test.go

WIN64DIR = x64
LINUX64DIR = linux64
OSX_INTELDIR = osx_intel
OSX_ARMDIR = osx_arm

smuggo:
	go fmt $(SOURCE)
	go build -o smuggo $(SOURCE)

build:
	GOOS=windows GOARCH=amd64 go build -o "$(WIN64DIR)/smuggo.exe" $(SOURCE)
	GOOS=linux GOARCH=amd64 go build -o "$(LINUX64DIR)/smuggo" $(SOURCE)
	GOOS=darwin GOARCH=amd64 go build -o "$(OSX_INTELDIR)/smuggo" $(SOURCE)
	GOOS=darwin GOARCH=arm64 go build -o "$(OSX_ARMDIR)/smuggo" $(SOURCE)

test:
	go fmt $(SOURCE) $(TEST)
	go test -test.v $(SOURCE) $(TEST)
