package main

import (
	"sync"
	"time"
)

// TODO: remember and use e-tag for re-downloads
// TODO: add disk cache

type httpDownloadCacheEntry struct {
	url          string
	data         []byte
	downloadedOn time.Time
}

// HTTPDownloadCache is a cache for http downloads
type HTTPDownloadCache struct {
	expirationTime time.Duration
	m              map[string]*httpDownloadCacheEntry
	mu             sync.Mutex
}

// NewHTTPDownloadCache creates HTTPDownloadCache
func NewHTTPDownloadCache() *HTTPDownloadCache {
	cache := &HTTPDownloadCache{
		expirationTime: time.Hour * 24,
		m:              map[string]*httpDownloadCacheEntry{},
	}
	// run a task to remove expired
	go func() {
		for {
			// sleep for a bit longer than exipration time
			time.Sleep(cache.expirationTime + time.Minute*4)
			cache.removeAllExpired()
		}
	}()
	return cache
}

func (c *HTTPDownloadCache) removeAllExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	var toRemove []string
	for k, e := range c.m {
		if time.Since(e.downloadedOn) > time.Hour*24 {
			toRemove = append(toRemove, k)
		}
	}
	for _, k := range toRemove {
		delete(c.m, k)
	}
}

// Download downloads url
func (c *HTTPDownloadCache) Download(uri string) ([]byte, bool, error) {
	c.mu.Lock()
	e := c.m[uri]
	c.mu.Unlock()

	// re-download after 24 hours
	if e != nil && time.Since(e.downloadedOn) < c.expirationTime {
		return e.data, true, nil
	}

	d, err := httpGet(uri)
	if err != nil {
		return nil, false, err
	}

	e = &httpDownloadCacheEntry{
		url:          uri,
		data:         d,
		downloadedOn: time.Now(),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[uri] = e
	return d, false, nil
}
