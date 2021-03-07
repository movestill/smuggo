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
	"errors"
	"fmt"
	"log"
	"os"
	"path"

	_ "github.com/mattn/go-sqlite3"
)

const imageTable = "images"
const imageTableVersion = 1
const imageTableHashIndexName = "images_hash_index"
const imageTableAblumKeyIndexName = "images_ablum_key_index"

const versionTable = "table_versions"

var imgTableCreateSQL = fmt.Sprintf(
	"CREATE TABLE %s (id INTEGER NOT NULL PRIMARY KEY, album_key TEXT, hash TEXT, filename TEXT);", imageTable)
var imgTableHashIndexSQL = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (hash)",
	imageTableHashIndexName, imageTable)
var imgTableAlbumKeyIndexSQL = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (album_key)",
	imageTableAblumKeyIndexName, imageTable)
var verTableCreateSQL = fmt.Sprintf(
	"CREATE TABLE %s (id INTEGER NOT NULL PRIMARY KEY, name TEXT, version INTEGER);", versionTable)

var imgTableInsertSQL = fmt.Sprintf("INSERT INTO %s (album_key, hash, filename) VALUES (?, ?, ?);", imageTable)
var imgTableDeleteSQL = fmt.Sprintf("DELETE FROM %s WHERE album_key = ?;", imageTable)
var imgTableGetDupesSQL = fmt.Sprintf("SELECT filename FROM %s WHERE album_key = ? AND hash = ?", imageTable)

// Ensure DB exists and is a compatible version.
func initDB() {
	dbFile := path.Join(smuggoDirFlag, "images.db")
	var createDb = false
	_, pathErr := os.Stat(dbFile)
	if pathErr != nil {
		createDb = true
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Error opening database %s: %q\n", dbFile, err)
	}
	defer db.Close()

	if createDb {
		createTables(db, imageTableVersion)
	}

	if err := validateTables(db); err != nil {
		log.Fatal(err)
	}
}

func openDB() *sql.DB {
	dbFile := path.Join(smuggoDirFlag, "images.db")

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Error opening database %s: %q\n", dbFile, err)
	}

	return db
}

// Create tables and indices for an empty DB.
// If more tables added, will need to rethink accepting a single version
// parameter.
func createTables(db *sql.DB, imgTableVersion int) {
	createSQL := fmt.Sprintf("%s\n%s\n%s", imgTableCreateSQL, verTableCreateSQL, imgTableHashIndexSQL)

	_, err := db.Exec(createSQL)
	if err != nil {
		log.Fatalf("Error creating database tables: %q\n", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	writeImgVersionSQL, err := tx.Prepare(
		fmt.Sprintf("INSERT INTO %s (name, version) VALUES (?, ?);", versionTable))
	if err != nil {
		log.Fatal(err)
	}

	defer writeImgVersionSQL.Close()
	_, err = writeImgVersionSQL.Exec(imageTable, imgTableVersion)
	if err != nil {
		log.Fatal(err)
	}

	tx.Commit()
}

func validateTables(db *sql.DB) error {
	rows, err := db.Query(fmt.Sprintf("SELECT name, version FROM %s;", versionTable))
	if err != nil {
		//fmt.Sprintf("Error validating table versions: %q\n", err)
		return err
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		var version int
		err = rows.Scan(&name, &version)
		if err != nil {
			return err
		}

		if name == imageTable && version != imageTableVersion {
			msg := fmt.Sprintf("Table: %s version is %d, but must be version %d", imageTable, version, imageTableVersion)
			return errors.New(msg)
		}
	}

	return nil
}

// Write image data for the given images to the DB.
func writeImageData(db *sql.DB, albumKey string, imgData []imageJSON) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	insertSQL, err := tx.Prepare(imgTableInsertSQL)
	if err != nil {
		log.Fatal(err)
	}

	defer insertSQL.Close()
	for _, row := range imgData {
		_, err = insertSQL.Exec(albumKey, row.ArchivedMD5, row.FileName)
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Fatalf("Failed to insert image data: %v, rollback failed: %v", err, rollbackErr)
			}
			log.Fatal(err)
		}
	}

	tx.Commit()
}

// Remove all image data for the given album.
func removeAlbumImages(db *sql.DB, albumKey string) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	deleteSQL, err := tx.Prepare(imgTableDeleteSQL)
	if err != nil {
		log.Fatal(err)
	}

	defer deleteSQL.Close()
	_, err = deleteSQL.Exec(albumKey)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			log.Fatalf("Failed deleting image data for album: %s: %v, rollback failed: %v\n",
				albumKey, err, rollbackErr)
		}
		log.Fatal(err)
	}

	tx.Commit()
}

// Get duplicates images from an album based on the given MD5 hash.
func getDuplicateImages(db *sql.DB, albumKey string, hash string) []string {
	filenames := make([]string, 0, 5)
	rows, err := db.Query(imgTableGetDupesSQL, albumKey, hash)
	if err != nil {
		log.Println("Error building query that checks for duplicate images")
		return filenames
	}

	defer rows.Close()
	for rows.Next() {
		var filename string
		err = rows.Scan(&filename)
		if err != nil {
			log.Println(err)
		}
		filenames = append(filenames, filename)
	}

	return filenames
}
