package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/fsnotify/fsnotify"
)

func main() {
	var (
		id     int
		space  string
		file   string
		now    bool
		silent bool
	)

	flag.IntVar(&id, "id", 0, "The id of the planet")
	flag.StringVar(&space, "space", "", "The spaceship cookie")
	flag.StringVar(&file, "file", "", "The map file")
	flag.BoolVar(&now, "now", false, "Skip checking for update, just update it now")
	flag.BoolVar(&silent, "silent", false, "Don't print that its updating")
	flag.Parse()

	if err := validateFlags(id, space, file); err != nil {
		fmt.Println(err)
		return
	}

	pmapFile, err := os.Open(file)
	if err != nil {
		fmt.Printf("upmap: error: %v\n", err)
		return
	}
	defer pmapFile.Close()

	if now {
		if !silent {
			fmt.Print("updating...")
		}
		if err := update(pmapFile, id, space); err != nil {
			fmt.Println(err)
			return
		}
		if !silent {
			fmt.Println("updated")
		}
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("upmap: error:", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(file)
	if err != nil {
		fmt.Println("upmap: error:", err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				if !silent {
					fmt.Print("updating...")
				}
				if err := update(pmapFile, id, space); err != nil {
					fmt.Println(err)
					continue
				}
				if !silent {
					fmt.Println("updated")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("upmap: fsnotify: error:", err)
		}
	}
}

func validateFlags(id int, space, file string) error {
	if file == "" {
		return fmt.Errorf("upmap: usage error: file is required")
	}
	if id == 0 {
		return fmt.Errorf("upmap: usage error: id is required")
	}
	if space == "" {
		return fmt.Errorf("upmap: usage error: space is required")
	}
	return nil
}

func readMapFile(pmapFile *os.File) ([]byte, error) {
	pmap, err := io.ReadAll(pmapFile)
	if err != nil {
		return nil, fmt.Errorf("upmap: error: %v", err)
	}
	return pmap, nil
}

func makeHTTPRequest(id int, space string, pmap []byte) error {
	// big boy form stuff
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.SetBoundary("---------------------------127549378316051060631914742458")
	writer.WriteField("nam", "")
	writer.WriteField("desc", "")
	writer.WriteField("thum", "")
	fileWriter, err := writer.CreateFormFile("map", "test.pmap")
	if err != nil {
		return fmt.Errorf("upmap: error: %v", err)
	}
	_, err = io.Copy(fileWriter, bytes.NewReader(pmap))
	if err != nil {
		return fmt.Errorf("upmap: error: %v", err)
	}
	writer.Close()

	url := fmt.Sprintf("https://www.planetarium.digital/games/edit/?id=%d", id)
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("upmap: error: %v", err)
	}

	req.Header.Set("Content-Type", "multipart/form-data; boundary=---------------------------127549378316051060631914742458")

	req.AddCookie(&http.Cookie{
		Name:   "spaceship",
		Value:  space,
		Path:   "/",
		Secure: true,
	})

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upmap: error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, resp.Status)
	}
	return nil
}

func update(pmapFile *os.File, id int, space string) error {
	pmap, err := readMapFile(pmapFile)
	if err != nil {
		return err
	}

	if err := makeHTTPRequest(id, space, pmap); err != nil {
		return err
	}

	return nil
}
