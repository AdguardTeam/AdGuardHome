package querylog

import (
	"fmt"
	"io"
	"os"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// QLogReader allows reading from multiple query log files in the reverse order.
//
// Please note that this is a stateful object.
// Internally, it contains a pointer to a particular query log file, and
// to a specific position in this file, and it reads lines in reverse order
// starting from that position.
type QLogReader struct {
	// qFiles - array with the query log files
	// The order is - from oldest to newest
	qFiles []*QLogFile

	currentFile int // Index of the current file
}

// NewQLogReader initializes a QLogReader instance
// with the specified files
func NewQLogReader(files []string) (*QLogReader, error) {
	qFiles := make([]*QLogFile, 0)

	for _, f := range files {
		q, err := NewQLogFile(f)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			// Close what we've already opened.
			cerr := closeQFiles(qFiles)
			if cerr != nil {
				log.Debug("querylog: closing files: %s", cerr)
			}

			return nil, err
		}

		qFiles = append(qFiles, q)
	}

	return &QLogReader{
		qFiles:      qFiles,
		currentFile: (len(qFiles) - 1),
	}, nil
}

// seekTS performs binary search of a query log record with the specified
// timestamp.  If the record is found, it sets QLogReader's position to point to
// that line, so that the next ReadNext call returned this line.
func (r *QLogReader) seekTS(timestamp int64) (err error) {
	for i := len(r.qFiles) - 1; i >= 0; i-- {
		q := r.qFiles[i]
		_, _, err = q.seekTS(timestamp)
		if err != nil {
			if errors.Is(err, ErrTSTooEarly) {
				// Look at the next file, since we've reached the end of this
				// one.  If there is no next file, it's not found.
				err = ErrTSNotFound

				continue
			} else if errors.Is(err, ErrTSTooLate) {
				// Just seek to the start then.  timestamp is probably between
				// the end of the previous one and the start of this one.
				return r.SeekStart()
			} else if errors.Is(err, ErrTSNotFound) {
				return err
			} else {
				return fmt.Errorf("seekts: file at index %d: %w", i, err)
			}
		}

		// The search is finished, and the searched element has been found.
		// Update currentFile only, position is already set properly in
		// QLogFile.
		r.currentFile = i

		return nil
	}

	if err != nil {
		return fmt.Errorf("seekts: %w", err)
	}

	return nil
}

// SeekStart changes the current position to the end of the newest file
// Please note that we're reading query log in the reverse order
// and that's why log start is actually the end of file
//
// Returns nil if we were able to change the current position.
// Returns error in any other case.
func (r *QLogReader) SeekStart() error {
	if len(r.qFiles) == 0 {
		return nil
	}

	r.currentFile = len(r.qFiles) - 1
	_, err := r.qFiles[r.currentFile].SeekStart()
	return err
}

// ReadNext reads the next line (in the reverse order) from the query log files.
// and shifts the current position left to the next (actually prev) line (or the next file).
// returns io.EOF if there's nothing to read more.
func (r *QLogReader) ReadNext() (string, error) {
	if len(r.qFiles) == 0 {
		return "", io.EOF
	}

	for r.currentFile >= 0 {
		q := r.qFiles[r.currentFile]
		line, err := q.ReadNext()
		if err != nil {
			// Shift to the older file
			r.currentFile--
			if r.currentFile < 0 {
				break
			}

			q = r.qFiles[r.currentFile]

			// Set it's position to the start right away
			_, err = q.SeekStart()

			// This is unexpected, return an error right away
			if err != nil {
				return "", err
			}
		} else {
			return line, nil
		}
	}

	// Nothing to read anymore
	return "", io.EOF
}

// Close closes the QLogReader
func (r *QLogReader) Close() error {
	return closeQFiles(r.qFiles)
}

// closeQFiles - helper method to close multiple QLogFile instances
func closeQFiles(qFiles []*QLogFile) error {
	var errs []error

	for _, q := range qFiles {
		err := q.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.List("error while closing QLogReader", errs...)
	}

	return nil
}
