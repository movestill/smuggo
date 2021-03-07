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
	db := setUpTestDB(t)
	defer db.Close()

	createTables(db, 0)

	if err := validateTables(db); err == nil {
		t.Error("Failed to report version mismatch")
	}
}

func TestWriteImageData(t *testing.T) {
	db := setUpTestDB(t)
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

func TestRetrieveHash(t *testing.T) {
	db := setUpTestDB(t)
	defer db.Close()

	createTables(db, imageTableVersion)

	albumKey1 := "fake-album-key"
	expHash := "fake-hash-1"
	expFilename := "img1.jpg"
	imgData := []imageJSON{{expHash, expFilename}, {"fake-hash-2", "img2.jpg"}}
	writeImageData(db, albumKey1, imgData)

	albumKey2 := "fake-album-key2"
	imgData2 := []imageJSON{{expHash, expFilename}}
	writeImageData(db, albumKey2, imgData2)

	rows, err := db.Query(
		fmt.Sprintf("SELECT album_key, hash, filename FROM %s WHERE hash = ?;", imageTable), expHash)
	if err != nil {
		t.Error(err)
	}

	var gotAlbumKey1 = false
	var gotAlbumKey2 = false
	var count = 0

	defer rows.Close()
	for rows.Next() {
		count++
		var actualAlbumKey string
		var actualHash string
		var actualFilename string
		err = rows.Scan(&actualAlbumKey, &actualHash, &actualFilename)
		if err != nil {
			t.Error(err)
		}
		if actualAlbumKey != albumKey1 && actualAlbumKey != albumKey2 {
			t.Errorf("Expected album key %s or %s, got %s\n", albumKey1, albumKey2, actualAlbumKey)
		}
		if actualAlbumKey == albumKey1 {
			gotAlbumKey1 = true
		} else if actualAlbumKey == albumKey2 {
			gotAlbumKey2 = true
		}
		if actualHash != expHash {
			t.Errorf("Expected hash %s, got %s\n", actualHash, expHash)
		}
		if actualFilename != expFilename {
			t.Errorf("Expected filename %s, got %s\n", actualFilename, expFilename)
		}
	}

	if !gotAlbumKey1 {
		t.Errorf("Did not get album key: %s", albumKey1)
	}
	if !gotAlbumKey2 {
		t.Errorf("Did not get album key: %s", albumKey2)
	}
	if count != 2 {
		t.Errorf("Expected 2 rows, but got %d rows", count)
	}
}

func TestRemoveAlbumImages(t *testing.T) {
	db := setUpTestDB(t)
	defer db.Close()

	createTables(db, imageTableVersion)

	albumKey1 := "fake-album-key"
	expHash := "fake-hash-1"
	expFilename := "img1.jpg"
	imgData := []imageJSON{{expHash, expFilename}, {"fake-hash-2", "img2.jpg"}}
	writeImageData(db, albumKey1, imgData)

	albumKey2 := "fake-album-key2"
	imgData2 := []imageJSON{{expHash, expFilename}}
	writeImageData(db, albumKey2, imgData2)

	removeAlbumImages(db, albumKey1)

	rows, err := db.Query(fmt.Sprintf("SELECT album_key FROM %s;", imageTable))
	if err != nil {
		t.Error(err)
	}

	defer rows.Close()
	for rows.Next() {
		var actualAlbumKey string
		err = rows.Scan(&actualAlbumKey)
		if err != nil {
			t.Error(err)
		}
		if actualAlbumKey != albumKey2 {
			t.Errorf("Expected album key %s, got %s\n", albumKey2, actualAlbumKey)
		}
	}
}

func TestGetDuplicateImages(t *testing.T) {
	db := setUpTestDB(t)
	defer db.Close()

	createTables(db, imageTableVersion)

	albumKey := "fake-album-key"
	expHash := "fake-hash-1"
	expFilename1 := "img1.jpg"
	expFilename2 := "img2.jpg"
	imgData := []imageJSON{{expHash, expFilename1}, {expHash, expFilename2}}
	writeImageData(db, albumKey, imgData)

	actualFilenames := getDuplicateImages(db, albumKey, expHash)
	if len(actualFilenames) != 2 {
		t.Errorf("Expected 2 filenames but got %d", len(actualFilenames))
	}

	var gotFilename1 = false
	var gotFilename2 = false
	for _, f := range actualFilenames {
		if f == expFilename1 {
			gotFilename1 = true
		}
		if f == expFilename2 {
			gotFilename2 = true
		}
	}

	if !gotFilename1 {
		t.Errorf("Did not get duplicate image: %s", expFilename1)
	}

	if !gotFilename2 {
		t.Errorf("Did not get duplicate image: %s", expFilename2)
	}
}

func setUpTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error("Error opening DB: ", err)
	}
	return db
}
