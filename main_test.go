package main

import (
	"bytes"
	"net/http"
	"os"
	"reflect"
	"testing"
)

func TestCacheFileOperations(t *testing.T) {
	// Setup: Ensure we don't accidentally use an existing cache file, 
	// and clean up after the test finishes.
	testFileName := "cache.json"
	os.Remove(testFileName)
	defer os.Remove(testFileName)

	// 1. Create mock data simulating a cached HTTP response
	mockCache := map[string]CacheEntry{
		"http://example.com/api/data": {
			Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Custom": []string{"Test"}},
			Body:       []byte(`{"status": "success", "data": [1, 2, 3]}`),
			StatusCode: 200,
		},
		"http://example.com/api/empty": {
			Header:     http.Header{},
			Body:       []byte(nil),
			StatusCode: 404,
		},
	}

	// 2. Test writing to the file
	err := writeToCache(mockCache)
	if err != nil {
		t.Fatalf("writeToCache failed: %v", err)
	}

	// Verify the file was actually created on disk
	if _, err := os.Stat(testFileName); os.IsNotExist(err) {
		t.Fatalf("Expected %s to be created, but it was not found", testFileName)
	}

	// 3. Test loading from the file into a new map
	loadedCache := make(map[string]CacheEntry)
	err = loadCache(&loadedCache)
	if err != nil {
		t.Fatalf("loadCache failed: %v", err)
	}

	// 4. Verify the loaded data matches the original mock data
	if len(loadedCache) != len(mockCache) {
		t.Fatalf("Expected cache map length %d, got %d", len(mockCache), len(loadedCache))
	}

	for key, originalEntry := range mockCache {
		loadedEntry, exists := loadedCache[key]
		if !exists {
			t.Errorf("Expected to find key %q in loaded cache, but it was missing", key)
			continue
		}

		if loadedEntry.StatusCode != originalEntry.StatusCode {
			t.Errorf("[%s] Expected status %d, got %d", key, originalEntry.StatusCode, loadedEntry.StatusCode)
		}

		if !bytes.Equal(loadedEntry.Body, originalEntry.Body) {
			t.Errorf("[%s] Expected body %s, got %s", key, string(originalEntry.Body), string(loadedEntry.Body))
		}

		// Ensure headers match exactly
		if !reflect.DeepEqual(loadedEntry.Header, originalEntry.Header) {
			t.Errorf("[%s] Expected headers %v, got %v", key, originalEntry.Header, loadedEntry.Header)
		}
	}
}

func TestLoadCache_FileDoesNotExist(t *testing.T) {
	// Setup: Ensure file does not exist
	os.Remove("cache.json")
	defer os.Remove("cache.json")

	emptyCache := make(map[string]CacheEntry)
	
	// If the file doesn't exist, loadCache should create it and return no error
	err := loadCache(&emptyCache)
	if err != nil {
		t.Fatalf("loadCache returned error when file didn't exist: %v", err)
	}

	if len(emptyCache) != 0 {
		t.Errorf("Expected empty cache, got %d items", len(emptyCache))
	}

	// Verify it created the fallback empty file
	if _, err := os.Stat("cache.json"); os.IsNotExist(err) {
		t.Fatal("loadCache was supposed to create an empty cache.json file, but didn't")
	}
}