package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	"hawx.me/code/hadfield"
)

const flickrURL = "https://api.flickr.com/services/rest/"

var grabAPIKey string
var grabUserID string

var cmdGrab = &hadfield.Command{
	Usage: "grab PHOTOPATH",
	Short: "grabs your user info, photos, and set from flickr",
	Long: `
  Grab retrieves your user info, photos, and sets from flickr and saves them
  to PHOTOPATH.

  Options:
    --api-key KEY      Your Flickr API key
    --user-id ID       Your Flickr user ID (with the @)
`,
	Run: func(cmd *hadfield.Command, args []string) {
		if len(args) != 1 {
			log.Fatal("Must provide PHOTOPATH")
		}

		if err := runGrab(args[0]); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	cmdGrab.Flag.StringVar(&grabAPIKey, "api-key", "", "")
	cmdGrab.Flag.StringVar(&grabUserID, "user-id", "", "")
}

func runGrab(root string) error {
	baseURL, _ := url.Parse(flickrURL)

	client := &httpClient{
		client:    http.DefaultClient,
		apiKey:    grabAPIKey,
		BaseURL:   baseURL,
		UserAgent: "me.hawx.flickr-mirror",
	}

	if err := grabUser(client, root); err != nil {
		return err
	}

	if err := grabPhotos(client, root); err != nil {
		log.Println(err)
	}

	if err := grabSets(client, root); err != nil {
		log.Println(err)
	}

	return nil
}

func grabUser(client *httpClient, root string) error {
	log.Println("user", grabUserID)

	resp, err := client.get("flickr.people.getInfo", url.Values{
		"user_id": {grabUserID},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var data struct {
		Person map[string]interface{} `json:"person"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	log.Println("  wrote user.json")
	return writeJSON(root, "user.json", data.Person)
}

func grabPhotos(client *httpClient, root string) error {
	page := 0
	pages := 10 // this will get set properly on the first loop

	for {
		if page > pages {
			return nil
		}

		resp, err := client.get("flickr.people.getPhotos", url.Values{
			"user_id":  {grabUserID},
			"per_page": {"100"},
			"page":     {strconv.Itoa(page)},
			"extras":   {"url_z,url_o"},
		})
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		page++

		var data struct {
			Photos struct {
				Page  int                      `json:"page"`
				Pages int                      `json:"pages"`
				Photo []map[string]interface{} `json:"photo"`
			} `json:"photos"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return err
		}
		pages = data.Photos.Pages

		for _, photo := range data.Photos.Photo {
			id := photo["id"].(string)
			log.Println("photo", id)

			dir := path.Join(root, "photos", id)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			infoResp, err := client.get("flickr.photos.getInfo", url.Values{
				"photo_id": {id},
			})
			if err != nil {
				return err
			}
			defer infoResp.Body.Close()

			var info struct {
				Photo map[string]interface{} `json:"photo"`
			}
			if err := json.NewDecoder(infoResp.Body).Decode(&info); err != nil {
				return err
			}

			if err := writeJSON(dir, "data.json", info.Photo); err != nil {
				return err
			}
			log.Println("  wrote data.json")

			exifResp, err := client.get("flickr.photos.getExif", url.Values{
				"photo_id": {id},
			})
			if err != nil {
				log.Println("no exif for", id)
			} else {
				defer exifResp.Body.Close()

				var exif struct {
					Photo map[string]interface{} `json:"photo"`
				}
				if err := json.NewDecoder(exifResp.Body).Decode(&exif); err != nil {
					return err
				}

				if err := writeJSON(dir, "exif.json", exif.Photo); err != nil {
					return err
				}
				log.Println("  wrote exif.json")
			}

			originalFilename := "photo_o." + info.Photo["originalformat"].(string)
			if err := writePhoto(dir, originalFilename, photo["url_o"].(string)); err != nil {
				return err
			}
			log.Println("  wrote", originalFilename)

			if err := writePhoto(dir, "photo_z.jpg", photo["url_z"].(string)); err != nil {
				return err
			}
			log.Println("  wrote photo_z.jpg")
		}
	}
}

func grabSets(client *httpClient, root string) error {
	page := 0
	pages := 10 // this will get set properly on the first loop

	for {
		if page > pages {
			return nil
		}

		resp, err := client.get("flickr.photosets.getList", url.Values{
			"user_id":  {grabUserID},
			"per_page": {"100"},
			"page":     {strconv.Itoa(page)},
		})
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		page++

		var data struct {
			Photosets struct {
				Page     int                      `json:"page"`
				Pages    int                      `json:"pages"`
				Photoset []map[string]interface{} `json:"photoset"`
			} `json:"photosets"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return err
		}
		pages = data.Photosets.Pages

		for _, set := range data.Photosets.Photoset {
			id := set["id"].(string)
			log.Println("set", id)

			dir := path.Join(root, "sets", id)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			idsResp, err := client.get("flickr.photosets.getPhotos", url.Values{
				"photoset_id": {set["id"].(string)},
			})
			if err != nil {
				return err
			}
			defer idsResp.Body.Close()

			var ids struct {
				Photoset struct {
					Photo []struct {
						ID string `json:"id"`
					} `json:"photo"`
				} `json:"photoset"`
			}
			if err := json.NewDecoder(idsResp.Body).Decode(&ids); err != nil {
				return err
			}

			var idList []string
			for _, photo := range ids.Photoset.Photo {
				idList = append(idList, photo.ID)
			}

			set["photoids"] = idList
			if err := writeJSON(dir, "data.json", set); err != nil {
				return err
			}
			log.Println("  wrote data.json")
		}
	}
}

func writeJSON(root, p string, v interface{}) error {
	file, err := os.Create(path.Join(root, p))
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writePhoto(root, p, u string) error {
	file, err := os.Create(path.Join(root, p))
	if err != nil {
		return err
	}

	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

type httpClient struct {
	client *http.Client
	apiKey string

	BaseURL   *url.URL
	UserAgent string
}

func (client *httpClient) get(method string, params url.Values) (*http.Response, error) {
	params.Add("method", method)
	params.Add("api_key", client.apiKey)
	params.Add("format", "json")
	params.Add("nojsoncallback", "1")

	reqURL := client.BaseURL.String() + "?" + params.Encode()

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", client.UserAgent)

	resp, err := client.client.Do(req)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("Received %d response", resp.StatusCode)
	}

	return resp, err
}
