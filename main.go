package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// приложение - розыгрыш
// При запросе анализируется User-Agent, и если это обычный браузер, то начинается розыгрыш

// суть - Рикролл
// Указывается вероятность и ссылка, на которую надо перенаправить (для реализации розыгрыша). Ссылок может быть несколько (для них указываются диапазон вероятностей)
// Действия для правильной ссылки: перенаправление на правильную ссылку, и предзагрузка html кода и его выдача.

// Never Gonna Give You Up https://www.youtube.com/watch?v=dQw4w9WgXcQ
// Call Me Maybe https://www.youtube.com/watch?v=fWNaR-rxAic

// для браузеров характерны указания ОС и Движков
// Mozilla
// Gecko
// Firefox
// Edge
// Safari
// Linux
// Windows
// MacOS

type RollItem struct {
	Start int    `json:"start,omitempty"`
	Stop  int    `json:"stop,omitempty"`
	URL   string `json:"url,omitempty"`
	Desc  string `json:"desc,omitempty"` //not use. for comment config
}

type Roll struct {
	Watch  string      `json:"watch,omitempty"`
	Target string      `json:"target,omitempty"`
	Method string      `json:"method,omitempty"` // prefetch, redirect(default)
	Rolls  []*RollItem `json:"rolls,omitempty"`
}

type Config []*Roll

func main() {
	configName := flag.String("conf", "config.json", "configuration file")
	flag.Parse()

	var config Config
	configFile, err := os.OpenFile(*configName, os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}
	configJson, err := io.ReadAll(configFile)
	if err != nil {
		panic(err)
	}
	configFile.Close()
	err = json.Unmarshal(configJson, &config)
	if err != nil {
		panic(err)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	preFetched := make(map[string][]byte)
	// TODO fetch header
	for _, c := range config {
		if c.Method == "prefetch" {
			resp, err := http.Get(c.Target)
			if err != nil || resp.StatusCode != 200 {
				c.Method = "redirect"
				log.Printf("Prefetched error (%v) with code:%v URL:%v. Method changed to redirect\n", err, resp.StatusCode, c.Target)
				continue
			}
			buf := &bytes.Buffer{}
			buf.ReadFrom(resp.Body)
			preFetched[c.Target] = buf.Bytes()
		}
	}

	http.ListenAndServe(":8080", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		r.URL.Scheme = ""
		url := r.Host + "/" + r.URL.String()
		url = strings.TrimRight(url, "/")
		log.Println(url)
		userAgent := r.Header.Get("User-Agent")
		var roll *Roll
		for _, rr := range config {
			if rr.Watch == url {
				roll = rr
				break
			}
		}
		if roll == nil {
			http.NotFound(rw, r)
			return
		}
		if strings.Contains(userAgent, "Mozilla") || strings.Contains(userAgent, "Gecko") {
			random := rnd.Intn(100)
			var item *RollItem
			for _, ri := range roll.Rolls {
				if ri.Start <= random && ri.Stop > random {
					item = ri
					break
				}
			}
			if item != nil {
				http.Redirect(rw, r, item.URL, http.StatusFound)
				return
			}
		}
		if data, has := preFetched[url]; has {
			rw.Write(data)
		} else {
			http.Redirect(rw, r, roll.Target, http.StatusFound)
		}
	}))
}
