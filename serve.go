package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	_ "github.com/mxk/go-sqlite/sqlite3"
	"hawx.me/code/hadfield"
	"hawx.me/code/route"
)

var cmdServe = &hadfield.Command{
	Usage: "serve PHOTOPATH",
	Short: "serves your photos",
	Long: `
  Serve takes your nicely indexed photos and shows them in a webapp.
`,
	Run: func(cmd *hadfield.Command, args []string) {
		if len(args) != 1 {
			log.Fatal("Must provide PHOTOPATH")
		}

		if err := runServe(args[0]); err != nil {
			log.Fatal(err)
		}
	},
}

type photosCtx struct {
	Photos   []photoRecord
	NextPage string
	PrevPage string
}

func runServe(root string) error {
	db, err := sql.Open("sqlite3", "db")
	if err != nil {
		return err
	}
	defer db.Close()

	templates, err := template.ParseGlob("templates/*.tmpl")
	if err != nil {
		return err
	}

	mux := route.New()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pageNo := 0
		if pageNo_, err := strconv.Atoi(r.FormValue("page")); err == nil {
			pageNo = pageNo_
		}

		photos, err := getPhotos(db, pageNo)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}

		w.Header().Set("Content-Type", "text/html")
		ctx := photosCtx{
			Photos:   photos,
			NextPage: "/?page=" + strconv.Itoa(pageNo+1),
		}

		if pageNo > 0 {
			ctx.PrevPage = "/?page=" + strconv.Itoa(pageNo-1)
		}

		err = templates.ExecuteTemplate(w, "photos.tmpl", ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	mux.HandleFunc("/photos/:photo", func(w http.ResponseWriter, r *http.Request) {
		photo, err := getPhoto(db, route.Vars(r)["photo"])
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = templates.ExecuteTemplate(w, "photo.tmpl", photo)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	})

	mux.HandleFunc("/photosets", func(w http.ResponseWriter, r *http.Request) {
		photosets, err := getPhotosets(db)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}

		w.Header().Set("Content-Type", "text/html")
		err = templates.ExecuteTemplate(w, "photosets.tmpl", photosets)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	mux.HandleFunc("/photosets/:photoset", func(w http.ResponseWriter, r *http.Request) {
		pageNo := 0
		if pageNo_, err := strconv.Atoi(r.FormValue("page")); err == nil {
			pageNo = pageNo_
		}

		photoset := route.Vars(r)["photoset"]

		photos, err := getPhotosInSet(db, photoset, pageNo)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}

		w.Header().Set("Content-Type", "text/html")
		ctx := photosCtx{
			Photos:   photos,
			NextPage: "/photosets/" + photoset + "/?page=" + strconv.Itoa(pageNo+1),
		}

		if pageNo > 0 {
			ctx.PrevPage = "/photosets/" + photoset + "/?page=" + strconv.Itoa(pageNo-1)
		}

		err = templates.ExecuteTemplate(w, "photos.tmpl", ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	mux.Handle("/photo/*path", http.StripPrefix("/photo/", http.FileServer(http.Dir(filepath.Join(root, "photos")))))

	mux.Handle("/public/*path", http.StripPrefix("/public", http.FileServer(http.Dir("public"))))

	log.Println("Serving at :8080")
	return http.ListenAndServe(":8080", mux)
}

type streamRecord struct {
	Id        string
	Username  string
	Realname  string
	PhotosUrl string
}

type photosetRecord struct {
	Id    string
	Title string
	Cover string
}

type photosetMemberRecord struct {
	Photoset string
	Photo    string
}

type photoRecord struct {
	Id    string
	Title string
}

func getPhotos(db *sql.DB, pageNo int) (records []photoRecord, err error) {
	rows, err := db.Query(`
    SELECT Id, Title
    FROM photo
    ORDER BY DateUploaded DESC
    LIMIT 10
    OFFSET ?`,
		pageNo*10)
	if err != nil {
		return records, err
	}
	defer rows.Close()

	for rows.Next() {
		var record photoRecord
		if err = rows.Scan(&record.Id, &record.Title); err != nil {
			return records, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func getPhoto(db *sql.DB, photo string) (record photoRecord, err error) {
	row := db.QueryRow("SELECT Id, Title FROM photo WHERE Id = ?",
		photo)

	err = row.Scan(&record.Id, &record.Title)
	return record, err
}

func getPhotosInSet(db *sql.DB, photoset string, pageNo int) (records []photoRecord, err error) {
	rows, err := db.Query(`
    SELECT photo.Id, photo.Title
    FROM photo
    INNER JOIN photoset_member ON photo.Id = photoset_member.Photo
    WHERE photoset_member.Photoset = ?
    ORDER BY DateUploaded DESC
    LIMIT 10
    OFFSET ?`,
		photoset,
		pageNo*10)
	if err != nil {
		return records, err
	}
	defer rows.Close()

	for rows.Next() {
		var record photoRecord
		if err = rows.Scan(&record.Id, &record.Title); err != nil {
			return records, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func getPhotosets(db *sql.DB) (records []photosetRecord, err error) {
	rows, err := db.Query("SELECT Id, Title, Cover FROM photoset")
	if err != nil {
		return records, err
	}
	defer rows.Close()

	for rows.Next() {
		var record photosetRecord
		if err = rows.Scan(&record.Id, &record.Title, &record.Cover); err != nil {
			return records, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}
