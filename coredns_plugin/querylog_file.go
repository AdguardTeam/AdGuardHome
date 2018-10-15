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

const enableGzip = false

func flushToFile(buffer []*logEntry) error {
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

	var zb bytes.Buffer
	filename := queryLogFileName

	// gzip enabled?
	if enableGzip {
		filename += ".gz"

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
	} else {
		zb = b
	}

	fileWriteLock.Lock()
	defer fileWriteLock.Unlock()
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("failed to create file \"%s\": %s", filename, err)
		return err
	}
	defer f.Close()

	n, err := f.Write(zb.Bytes())
	if err != nil {
		log.Printf("Couldn't write to file: %s", err)
		return err
	}

	log.Printf("ok \"%s\": %v bytes written", filename, n)

	return nil
}

func checkBuffer(buffer []*logEntry, b bytes.Buffer) error {
	l := len(buffer)
	d := json.NewDecoder(&b)

	i := 0
	for d.More() {
		entry := &logEntry{}
		err := d.Decode(entry)
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
	from := queryLogFileName
	to := queryLogFileName + ".1"

	if enableGzip {
		from = queryLogFileName + ".gz"
		to = queryLogFileName + ".gz.1"
	}

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
	files := []string{}

	if enableGzip {
		files = []string{
			queryLogFileName + ".gz",
			queryLogFileName + ".gz.1",
		}
	} else {
		files = []string{
			queryLogFileName,
			queryLogFileName + ".1",
		}
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

		f, err := os.Open(file)
		if err != nil {
			log.Printf("Failed to open file \"%s\": %s", file, err)
			// try next file
			continue
		}
		defer f.Close()

		var d *json.Decoder

		if enableGzip {
			trace("Creating gzip reader")
			zr, err := gzip.NewReader(f)
			if err != nil {
				log.Printf("Failed to create gzip reader: %s", err)
				continue
			}
			defer zr.Close()

			trace("Creating json decoder")
			d = json.NewDecoder(zr)
		} else {
			d = json.NewDecoder(f)
		}

		i := 0
		over := 0
		max := 10000 * time.Second
		var sum time.Duration
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
				// trace("skipping entry") // debug logging
				continue
			}

			if entry.Elapsed > max {
				over++
			} else {
				sum += entry.Elapsed
			}

			i++
			err = onEntry(&entry)
			if err != nil {
				return err
			}
		}
		elapsed := time.Since(now)
		var perunit time.Duration
		var avg time.Duration
		if i > 0 {
			perunit = elapsed / time.Duration(i)
			avg = sum / time.Duration(i)
		}
		log.Printf("file \"%s\": read %d entries in %v, %v/entry, %v over %v, %v avg", file, i, elapsed, perunit, over, max, avg)
	}
	return nil
}

func appendFromLogFile(values []*logEntry, maxLen int, timeWindow time.Duration) []*logEntry {
	a := []*logEntry{}

	onEntry := func(entry *logEntry) error {
		a = append(a, entry)
		if len(a) > maxLen {
			toskip := len(a) - maxLen
			a = a[toskip:]
		}
		return nil
	}

	needMore := func() bool {
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
