// Package hn implements a really basic Hacker News clientService
package hn

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	apiBase = "https://hacker-news.firebaseio.com/v0"
)

// clientService is an API clientService used to interact with the Hacker News API
type clientService struct {
	apiBase string
	cache   storyCache
}

// Making the clientService zero value useful without forcing users to do something
// like `NewClientService()`
func (c *clientService) defaultify() {
	if c.apiBase == "" {
		c.apiBase = apiBase
		c.cache = cache
	}
}

// TopItems returns the ids of roughly 450 top items in decreasing order. These
// should map directly to the top 450 things you would see on HN if you visited
// their site and kept going to the next page.
//
// TopItmes does not filter out job listings or anything else, as the type of
// each item is unknown without further API calls.
func (c *clientService) topItems() ([]int, error) {
	c.defaultify()
	resp, err := http.Get(fmt.Sprintf("%s/topstories.json", c.apiBase))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var ids []int
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (c *clientService) getItem(id int) (Item, error) {
	c.defaultify()
	var item Item

	cached := c.cache.Get(id)

	if cached != nil {
		item = *cached
		item.FromCache = true
		return item, nil
	}

	resp, err := http.Get(fmt.Sprintf("%s/item/%d.json", c.apiBase, id))

	if err != nil {
		return item, err
	}

	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&item)

	if err != nil {
		return item, err
	}

	c.cache.Add(id, &item)

	return item, nil
}

// Item represents a single item returned by the HN API. This can have a type
// of "story", "comment", or "job" (and probably more values), and one of the
// URL or Text fields will be set, but not both.
//
// For the purpose of this exercise, we only care about items where the
// type is "story", and the URL is set.
type Item struct {
	By          string `json:"by"`
	Descendants int    `json:"descendants"`
	ID          int    `json:"id"`
	Kids        []int  `json:"kids"`
	Score       int    `json:"score"`
	Time        int    `json:"time"`
	Title       string `json:"title"`
	Type        string `json:"type"`

	// Only one of text and URL should exist
	Text      string `json:"text"`
	URL       string `json:"url"`
	Position  int
	err       error
	FromCache bool
}

func (i Item) IsStoryLink() bool {
	return i.Type == "story" && i.URL != ""
}

func (i Item) Error() error {
	return i.err
}

type Client struct {
	storyCount    int
	maxConcurrent int
	service       clientService
	initialized   bool
}

func (c *Client) getItem(storyId int, position int, out chan<- Item) {
	item, err := c.service.getItem(storyId)

	item.Position = position

	if err != nil {
		item.err = err
	}

	out <- item
}

func (c *Client) defaultify() {
	if !c.initialized {
		c.maxConcurrent = 10
		c.storyCount = 30
		c.service = clientService{}
		c.initialized = true
	}
}

func (c *Client) Fill(storyList *[]Item) error {
	c.defaultify()

	storyIds, err := c.service.topItems()

	if err != nil {
		return err
	}

	currRequests := 0

	bucket := make([]Item, c.maxConcurrent)

	ch := make(chan Item, c.maxConcurrent)

	full := false

	for storyNumber, id := range storyIds {
		if full {
			break
		}

		if currRequests < c.maxConcurrent {
			go c.getItem(id, storyNumber, ch)
			currRequests++
			continue
		}

		for currRequests > 0 {
			storyItem := <-ch

			bucket[storyItem.Position%c.maxConcurrent] = storyItem

			currRequests--
		}

		for _, storyItem := range bucket {
			// if story item is errored out, just skip it
			if storyItem.Error() != nil {
				fmt.Printf("%+v", storyItem.Error())
				continue
			}

			if storyItem.IsStoryLink() {
				*storyList = append(*storyList, storyItem)
			}

			if len(*storyList) >= c.storyCount {
				full = true
				break
			}
		}

		// now that bucket is empty, we still need to request the story that
		// corresponds to the current id in the outer range loop
		go c.getItem(id, storyNumber, ch)
		currRequests++
	}

	return nil
}
