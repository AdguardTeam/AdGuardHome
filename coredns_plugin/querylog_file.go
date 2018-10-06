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

const (
	queryLogRotationPeriod = time.Hour * 24  // rotate the log every 24 hours
	queryLogFileName       = "querylog.json" // .gz added during compression
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

func periodicQueryLogRotate(t time.Duration) {
	for range time.Tick(t) {
		err := rotateQueryLog()
		if err != nil {
			log.Printf("Failed to rotate querylog: %s", err)
			// do nothing, continue rotating
		}
	}
}
