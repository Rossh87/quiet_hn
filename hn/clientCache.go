package hn

import (
	"container/list"
	"sync"
	"time"
)

type storyCache struct {
	mutex    sync.Mutex
	dq       list.List
	idx      map[int]*list.Element
	capacity int
}

func (sc *storyCache) Get(id int) *Item {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if existing, ok := sc.idx[id]; ok {
		sc.dq.MoveToFront(existing)
		i, ok := existing.Value.(*Item)

		if !ok {
			panic("found element in story cache whose value was not of type 'Item'")
		}

		return i
	}

	return nil
}

func (sc *storyCache) Add(id int, i *Item) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if sc.dq.Len() >= sc.capacity {
		oldest := sc.dq.Back()

		oldId := oldest.Value.(*Item).ID

		sc.dq.Remove(oldest)

		delete(sc.idx, oldId)
	}

	el := sc.dq.PushFront(i)

	sc.idx[id] = el
}

var cache = storyCache{
	capacity: 100,
	idx:      make(map[int]*list.Element),
}

type idCache struct {
	refreshing bool
	ttl        time.Duration
	expBuffer  time.Duration
	exp        time.Time
	mutex      sync.Mutex
	ids        []int
}

type idCacheResult struct {
	shouldRefresh bool
	ids           []int
}

func (c *idCache) Get() idCacheResult {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var result idCacheResult

	if time.Until(c.exp) <= 0 {
		c.ids = nil
		return result
	}

	result.ids = c.ids

	if time.Until(c.exp) <= c.expBuffer && !c.refreshing {
		result.shouldRefresh = true
		// prevent subsequent Get requests that occur in the refresh window
		// from triggering a refresh before the first refresh can complete.
		c.refreshing = true
	}

	return result
}

func (c *idCache) Add(ids []int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.exp = time.Now().Add(c.ttl)
	c.ids = ids
	c.refreshing = false
}

var itemIdCache = idCache{
	ttl:       10 * time.Second,
	expBuffer: 5 * time.Second,
}
