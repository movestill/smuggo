// Copyright 2021 Timothy Gion
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
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestVersionMismatch(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	defer db.Close()

	createTables(db, 0)

	if err := validateTables(db); err == nil {
		t.Error("Failed to report version mismatch")
	}
}

func TestWriteImageData(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	defer db.Close()

	createTables(db, imageTableVersion)

	albumKey := "fake-album-key"
	imgData := []imageJSON{{"fake-hash-1", "img1.jpg"}, {"fake-hash-2", "img2.jpg"}}
	writeImageData(db, albumKey, imgData)

	rows, err := db.Query(fmt.Sprintf("SELECT album_key, hash, filename FROM %s;", imageTable))
	if err != nil {
		t.Error(err)
	}

	defer rows.Close()
	var ind = 0
	for rows.Next() {
		var actualAlbumKey string
		var actualHash string
		var actualFilename string
		err = rows.Scan(&actualAlbumKey, &actualHash, &actualFilename)
		if err != nil {
			t.Error(err)
		}
		if actualAlbumKey != albumKey {
			t.Errorf("Expected album key %s, got %s\n", albumKey, actualAlbumKey)
		}
		if actualHash != imgData[ind].ArchivedMD5 {
			t.Errorf("Expected hash %s, got %s\n", actualHash, imgData[ind].ArchivedMD5)
		}
		if actualFilename != imgData[ind].FileName {
			t.Errorf("Expected filename %s, got %s\n", actualFilename, imgData[ind].FileName)
		}
		ind++
	}
}
