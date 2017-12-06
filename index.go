package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mxk/go-sqlite/sqlite3"
	"github.com/pkg/errors"
	"hawx.me/code/hadfield"
)

var cmdIndex = &hadfield.Command{
	Usage: "index PHOTOPATH",
	Short: "indexes your photos",
	Long: `
  Index takes a folder of photo+json and creates a sqlite3 database for quicker access.
`,
	Run: func(cmd *hadfield.Command, args []string) {
		if len(args) != 1 {
			log.Fatal("Must provide PHOTOPATH")
		}

		if err := runIndex(args[0]); err != nil {
			log.Fatal(err)
		}
	},
}

type userData struct {
	Id        string `json:"id"`
	Username  string `json:"username"`
	Realname  string `json:"realname"`
	PhotosUrl string `json:"photosurl"`
}

type setData struct {
	Id      string   `json:"id"`
	Title   string   `json:"title"`
	Primary string   `json:"primary"`
	Photos  []string `json:"photoids"`
}

type photoData struct {
	Id           string `json:"id"`
	Title        string `json:"title"`
	DateUploaded int    `json:"dateuploaded,string"`
}

func runIndex(root string) error {
	db, err := sql.Open("sqlite3", "db")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
    CREATE TABLE stream (
      Id        TEXT PRIMARY KEY,
      Username  TEXT,
      Realname  TEXT,
      PhotosUrl TEXT
    );

    CREATE TABLE photoset (
      Id    TEXT PRIMARY KEY,
      Title TEXT,
      Cover TEXT
    );

    CREATE TABLE photoset_member (
      Photoset TEXT,
      Photo    TEXT,
      FOREIGN KEY(Photoset) REFERENCES photoset(Id),
      FOREIGN KEY(Photo)    REFERENCES photo(Id)
    );

    CREATE TABLE photo (
      Id    TEXT PRIMARY KEY,
      Title TEXT,
      DateUploaded INTEGER
    );
  `)
	if err != nil {
		return err
	}

	{
		log.Println("reading user data")
		// read `root/data.json` as `userData` and put in `stream` table
		file, err := os.Open(filepath.Join(root, "data.json"))
		if err != nil {
			return err
		}
		defer file.Close()

		var v userData
		if err = json.NewDecoder(file).Decode(&v); err != nil {
			return err
		}

		_, err = db.Exec(`INSERT INTO stream(Id, Username, Realname, PhotosUrl) VALUES (?, ?, ?, ?)`,
			v.Id, v.Username, v.Realname, v.PhotosUrl)
		if err != nil {
			return err
		}
	}

	log.Println("reading sets")
	// read `root/sets/*/data.json` as `setData` and put in `set` table and `set_photo` table
	err = filepath.Walk(filepath.Join(root, "sets"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) == "data.json" {
			file, err := os.Open(path)
			if err != nil {
				return errors.WithMessage(err, path)
			}
			defer file.Close()

			var v setData
			if err = json.NewDecoder(file).Decode(&v); err != nil {
				return errors.WithMessage(err, path)
			}

			_, err = db.Exec(`INSERT INTO photoset(Id, Title, Cover) VALUES (?, ?, ?)`,
				v.Id, v.Title, v.Primary)
			if err != nil {
				return errors.WithMessage(err, path)
			}

			for _, photo := range v.Photos {
				_, err = db.Exec(`INSERT INTO photoset_member(Photoset, Photo) VALUES (?, ?)`,
					v.Id, photo)
				if err != nil {
					return errors.WithMessage(err, path)
				}
			}

			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Println("reading photos")
	// read `root/photos/*/data.json` as `photoData` and put in `photo` table
	return filepath.Walk(filepath.Join(root, "photos"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) == "data.json" {
			file, err := os.Open(path)
			if err != nil {
				return errors.WithMessage(err, path)
			}
			defer file.Close()

			var v photoData
			if err = json.NewDecoder(file).Decode(&v); err != nil {
				return errors.WithMessage(err, path)
			}

			_, err = db.Exec(`INSERT INTO photo(Id, Title, DateUploaded) VALUES (?, ?, ?)`,
				v.Id, v.Title, v.DateUploaded)
			if err != nil {
				return errors.WithMessage(err, path)
			}

			return filepath.SkipDir
		}

		return nil
	})
}
