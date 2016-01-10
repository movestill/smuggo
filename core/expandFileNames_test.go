package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testDir = "expandTestDir"

var testFileNames = []string{
	"milk.jpg", "orange.png", "star.png", "rose.jpg"}

func TestPassThru(t *testing.T) {
	passThru := func(pattern string) (matches []string, err error) {
		matches = []string{pattern}
		err = nil
		return
	}

	filenames := []string{"see.png", "face.jpg", "orange.jpg"}
	expected := []string{"see.png", "face.jpg", "orange.jpg"}
	actual := expandFileNames(filenames, passThru)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}
}

func TestDoubled(t *testing.T) {
	doubled := func(pattern string) (matches []string, err error) {
		matches = []string{pattern, pattern}
		err = nil
		return
	}

	filenames := []string{"see.png", "face.jpg", "orange.jpg"}
	expected := []string{"see.png", "see.png", "face.jpg", "face.jpg",
		"orange.jpg", "orange.jpg"}
	actual := expandFileNames(filenames, doubled)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}
}

func TestStar(t *testing.T) {
	expected := []string{
		testDir + "/milk.jpg",
		testDir + "/orange.png",
		testDir + "/rose.jpg",
		testDir + "/star.png"}
	actual := expandFileNames([]string{testDir + "/*"}, filepath.Glob)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}
}

func TestStarJpg(t *testing.T) {
	expected := []string{testDir + "/milk.jpg", testDir + "/rose.jpg"}
	actual := expandFileNames([]string{testDir + "/*.jpg"}, filepath.Glob)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}
}

func TestStarPng(t *testing.T) {
	expected := []string{testDir + "/orange.png", testDir + "/star.png"}
	actual := expandFileNames([]string{testDir + "/*.png"}, filepath.Glob)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}
}

func TestMain(m *testing.M) {
	if err := os.Mkdir(testDir, os.ModeDir|0755); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	for _, name := range testFileNames {
		f, err := os.Create(testDir + "/" + name)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		f.Close()
	}

	exitCode := m.Run()

	if err := os.RemoveAll(testDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	os.Exit(exitCode)
}
