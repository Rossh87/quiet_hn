package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rossh87/quiet_hn/hn"
)

func main() {
	// parse flags
	var port, numStories int
	flag.IntVar(&port, "port", 3000, "the port to start the web server on")
	flag.IntVar(&numStories, "num_stories", 30, "the number of top stories to display")
	flag.Parse()

	tpl := template.Must(template.ParseFiles("./index.gohtml"))

	http.HandleFunc("/", handler(numStories, tpl))

	// Start the server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handler(numStories int, tpl *template.Template) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var client hn.Client
		ids, err := client.TopItems()
		if err != nil {
			http.Error(w, "Failed to load top stories", http.StatusInternalServerError)
			return
		}
		var stories []item
		launchCount := 0
		bucket := make([]item, 5)
		c := make(chan hn.Item, 5)
		storiesFull := false
		for order, id := range ids {
			if storiesFull {
				break
			}

			if launchCount < 5 {
				go client.GetItem(id, order, c)
				launchCount++
			} else {
				for launchCount > 0 {
					hnItem := <-c
					item := parseHNItem(hnItem)
					bucket[item.Order%5] = item
					launchCount--
				}
				for _, item := range bucket {
					if item.Error() != nil {
						fmt.Printf("%+v", item.Error())
						continue
					}

					if isStoryLink(item) {
						stories = append(stories, item)
						if len(stories) >= numStories {
							storiesFull = true
							break
						}
					}
				}
			}
			// hnItem, err := client.GetItem(id)
			// if err != nil {
			// 	continue
			// }
			// item := parseHNItem(hnItem)
			// if isStoryLink(item) {
			// 	stories = append(stories, item)
			// 	if len(stories) >= numStories {
			// 		break
			// 	}
			// }
		}
		data := templateData{
			Stories: stories,
			Time:    time.Now().Sub(start),
		}
		err = tpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Failed to process the template", http.StatusInternalServerError)
			return
		}
	})
}

func isStoryLink(item item) bool {
	return item.Type == "story" && item.URL != ""
}

func parseHNItem(hnItem hn.Item) item {
	ret := item{Item: hnItem}
	url, err := url.Parse(ret.URL)
	if err == nil {
		ret.Host = strings.TrimPrefix(url.Hostname(), "www.")
	}
	return ret
}

// item is the same as the hn.Item, but adds the Host field
type item struct {
	hn.Item
	Host string
}

type templateData struct {
	Stories []item
	Time    time.Duration
}
