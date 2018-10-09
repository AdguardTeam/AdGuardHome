package dnsfilter

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-test/deep"
)

var (
	fileWriteLock sync.Mutex
)

func flushToFile(buffer []logEntry) error {
	if len(buffer) == 0 {
		return nil
	}
	start := time.Now()

	var b bytes.Buffer
	e := json.NewEncoder(&b)
	for _, entry := range buffer {
		err := e.Encode(entry)
		if err != nil {
			log.Printf("Failed to marshal entry: %s", err)
			return err
		}
	}

	elapsed := time.Since(start)
	log.Printf("%d elements serialized via json in %v: %d kB, %v/entry, %v/entry", len(buffer), elapsed, b.Len()/1024, float64(b.Len())/float64(len(buffer)), elapsed/time.Duration(len(buffer)))

	err := checkBuffer(buffer, b)
	if err != nil {
		log.Printf("failed to check buffer: %s", err)
		return err
	}

	filenamegz := queryLogFileName + ".gz"

	var zb bytes.Buffer

	zw := gzip.NewWriter(&zb)
	zw.Name = queryLogFileName
	zw.ModTime = time.Now()

	_, err = zw.Write(b.Bytes())
	if err != nil {
		log.Printf("Couldn't compress to gzip: %s", err)
		zw.Close()
		return err
	}

	if err = zw.Close(); err != nil {
		log.Printf("Couldn't close gzip writer: %s", err)
		return err
	}

	fileWriteLock.Lock()
	defer fileWriteLock.Unlock()
	f, err := os.OpenFile(filenamegz, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("failed to create file \"%s\": %s", filenamegz, err)
		return err
	}
	defer f.Close()

	n, err := f.Write(zb.Bytes())
	if err != nil {
		log.Printf("Couldn't write to file: %s", err)
		return err
	}

	log.Printf("ok \"%s\": %v bytes written", filenamegz, n)

	return nil
}

func checkBuffer(buffer []logEntry, b bytes.Buffer) error {
	l := len(buffer)
	d := json.NewDecoder(&b)

	i := 0
	for d.More() {
		var entry logEntry
		err := d.Decode(&entry)
		if err != nil {
			log.Printf("Failed to decode: %s", err)
			return err
		}
		if diff := deep.Equal(entry, buffer[i]); diff != nil {
			log.Printf("decoded buffer differs: %s", diff)
			return fmt.Errorf("decoded buffer differs: %s", diff)
		}
		i++
	}
	if i != l {
		err := fmt.Errorf("check fail: %d vs %d entries", l, i)
		log.Print(err)
		return err
	}
	log.Printf("check ok: %d entries", i)

	return nil
}

func rotateQueryLog() error {
	from := queryLogFileName + ".gz"
	to := queryLogFileName + ".gz.1"

	if _, err := os.Stat(from); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		return nil
	}

	err := os.Rename(from, to)
	if err != nil {
		log.Printf("Failed to rename querylog: %s", err)
		return err
	}

	log.Printf("Rotated from %s to %s successfully", from, to)

	return nil
}

func periodicQueryLogRotate() {
	for range time.Tick(queryLogRotationPeriod) {
		err := rotateQueryLog()
		if err != nil {
			log.Printf("Failed to rotate querylog: %s", err)
			// do nothing, continue rotating
		}
	}
}

func genericLoader(onEntry func(entry *logEntry) error, needMore func() bool, timeWindow time.Duration) error {
	now := time.Now()
	// read from querylog files, try newest file first
	files := []string{
		queryLogFileName + ".gz",
		queryLogFileName + ".gz.1",
	}

	// read from all files
	for _, file := range files {
		if !needMore() {
			break
		}
		if _, err := os.Stat(file); os.IsNotExist(err) {
			// do nothing, file doesn't exist
			continue
		}

		trace("Opening file %s", file)
		f, err := os.Open(file)
		if err != nil {
			log.Printf("Failed to open file \"%s\": %s", file, err)
			// try next file
			continue
		}
		defer f.Close()

		trace("Creating gzip reader")
		zr, err := gzip.NewReader(f)
		if err != nil {
			log.Printf("Failed to create gzip reader: %s", err)
			continue
		}
		defer zr.Close()

		trace("Creating json decoder")
		d := json.NewDecoder(zr)

		i := 0
		// entries on file are in oldest->newest order
		// we want maxLen newest
		for d.More() {
			if !needMore() {
				break
			}
			var entry logEntry
			err := d.Decode(&entry)
			if err != nil {
				log.Printf("Failed to decode: %s", err)
				// next entry can be fine, try more
				continue
			}

			if now.Sub(entry.Time) > timeWindow {
				trace("skipping entry")
				continue
			}

			i++
			err = onEntry(&entry)
			if err != nil {
				return err
			}
		}
		elapsed := time.Since(now)
		log.Printf("file \"%s\": read %d entries in %v, %v/entry", file, i, elapsed, elapsed/time.Duration(i))
	}
	return nil
}

func appendFromLogFile(values []logEntry, maxLen int, timeWindow time.Duration) []logEntry {
	a := []logEntry{}

	onEntry := func(entry *logEntry) error {
		a = append(a, *entry)
		if len(a) > maxLen {
			toskip := len(a) - maxLen
			a = a[toskip:]
		}
		return nil
	}

	needMore := func() bool {
		if len(a) >= maxLen {
			return false
		}
		return true
	}

	err := genericLoader(onEntry, needMore, timeWindow)
	if err != nil {
		log.Printf("Failed to load entries from querylog: %s", err)
		return values
	}

	// now that we've read all eligible entries, reverse the slice to make it go from newest->oldest
	for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
		a[left], a[right] = a[right], a[left]
	}

	// append it to values
	values = append(values, a...)

	// then cut off of it is bigger than maxLen
	if len(values) > maxLen {
		values = values[:maxLen]
	}

	return values
}
