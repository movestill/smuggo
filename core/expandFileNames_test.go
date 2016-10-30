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
