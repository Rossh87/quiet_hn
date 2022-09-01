package hn

import (
	"container/list"
	"sync"
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
	capacity: 50,
	idx:      make(map[int]*list.Element),
}
