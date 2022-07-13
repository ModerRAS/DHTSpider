package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"time"

	"encoding/base64"

	"github.com/gogs/chardet"
	"github.com/shiyanhui/dht"
	"golang.org/x/text/encoding/ianaindex"
)

type file struct {
	Path   []interface{} `json:"path"`
	Length int           `json:"length"`
}

type bitTorrent struct {
	InfoHash          string `json:"infohash"`
	Name              string `json:"name"`
	Files             []file `json:"files,omitempty"`
	Length            int    `json:"length,omitempty"`
	RawMetaDataBase64 string `json:"rawmetadatabase64,omitempty"`
	GetDateTime       int64  `json:"getdatetime,omitempty"`
}

func ConvertToUTF8(data []byte) string {
	fmt.Println("Convert To UTF-8!")
	detector := chardet.NewTextDetector()
	result, _ := detector.DetectBest(data)
	fmt.Printf("Detect Charset is %s, %s\n\n", result.Charset, result.Language)
	e, _ := ianaindex.IANA.Encoding(result.Charset)
	decoder := e.NewDecoder()
	str, _ := decoder.Bytes(data)
	return string(str)
}

func main() {
	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	w := dht.NewWire(65536, 1024, 256)

	go func() {
		for resp := range w.Response() {
			metadata, err := dht.Decode(resp.MetadataInfo)

			if err != nil {
				continue
			}
			info := metadata.(map[string]interface{})

			if _, ok := info["name"]; !ok {
				continue
			}

			bt := bitTorrent{
				InfoHash:          hex.EncodeToString(resp.InfoHash),
				Name:              info["name"].(string),
				RawMetaDataBase64: base64.StdEncoding.EncodeToString([]byte(resp.MetadataInfo)),
				GetDateTime:       time.Now().Unix(),
			}

			if v, ok := info["files"]; ok {
				files := v.([]interface{})
				bt.Files = make([]file, len(files))

				for i, item := range files {
					f := item.(map[string]interface{})
					// var paths []string = make([]string, len(f["path"].([]interface{})))
					// for j, jitem := range f["path"].([]interface{}) {
					// 	paths[j] = ConvertToUTF8(jitem.([]byte))
					// }
					bt.Files[i] = file{
						Path:   f["path"].([]interface{}),
						Length: f["length"].(int),
					}
				}
			} else if _, ok := info["length"]; ok {
				bt.Length = info["length"].(int)
			}

			data, err := json.Marshal(bt)
			if err == nil {
				fmt.Printf("%s: InfoHash: %s, Name: \n\n", time.Now().Format("2006-01-02 15:04:05"), bt.InfoHash, bt.Name)
				// f, ferr := os.Create("torrent/" + bt.Name + "-" + bt.InfoHash + ".json")
				// if ferr == nil {
				// 	f.Write(data)
				// 	f.Close()
				// }
				// g, ferr := os.Create("bencode/" + bt.Name + "-" + bt.InfoHash + ".bencode")
				// if ferr == nil {
				// 	g.Write(resp.MetadataInfo)
				// 	g.Close()
				// }
				resp, resperr := http.Post("https://post.add", "application/json", bytes.NewBuffer(data))

				if resperr != nil {
					panic(resperr)
				}

				defer resp.Body.Close()

				if resp.StatusCode == http.StatusCreated {
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						panic(err)
					}
					jsonStr := string(body)
					fmt.Println("Response: ", jsonStr)

				} else {
					fmt.Println("Get failed with error: ", resp.Status)
				}
			}
		}
	}()
	go w.Run()

	config := dht.NewCrawlConfig()
	config.Address = ":25535"
	config.OnAnnouncePeer = func(infoHash, ip string, port int) {
		w.Request([]byte(infoHash), ip, port)
	}
	d := dht.New(config)

	d.Run()
}
