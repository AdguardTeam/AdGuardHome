package querylog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// qLogReader allows reading from multiple query log files in the reverse
// order.
//
// Please note that this is a stateful object.  Internally, it contains a
// pointer to a particular query log file, and to a specific position in this
// file, and it reads lines in reverse order starting from that position.
type qLogReader struct {
	// logger is used for logging the operation of the query log reader.  It
	// must not be nil.
	logger *slog.Logger

	// qFiles is an array with the query log files.  The order is from oldest
	// to newest.
	qFiles []*qLogFile

	// currentFile is the index of the current file.
	currentFile int
}

// newQLogReader initializes a qLogReader instance with the specified files.
func newQLogReader(ctx context.Context, logger *slog.Logger, files []string) (*qLogReader, error) {
	qFiles := make([]*qLogFile, 0)

	for _, f := range files {
		q, err := newQLogFile(f)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			// Close what we've already opened.
			cErr := closeQFiles(qFiles)
			if cErr != nil {
				logger.DebugContext(ctx, "closing files", slogutil.KeyError, cErr)
			}

			return nil, err
		}

		qFiles = append(qFiles, q)
	}

	return &qLogReader{
		logger:      logger,
		qFiles:      qFiles,
		currentFile: len(qFiles) - 1,
	}, nil
}

// seekTS performs binary search of a query log record with the specified
// timestamp.  If the record is found, it sets qLogReader's position to point
// to that line, so that the next ReadNext call returned this line.
func (r *qLogReader) seekTS(ctx context.Context, timestamp int64) (err error) {
	for i := len(r.qFiles) - 1; i >= 0; i-- {
		q := r.qFiles[i]
		_, _, err = q.seekTS(ctx, r.logger, timestamp)
		if err != nil {
			if errors.Is(err, errTSTooEarly) {
				// Look at the next file, since we've reached the end of this
				// one.  If there is no next file, it's not found.
				err = errTSNotFound

				continue
			} else if errors.Is(err, errTSTooLate) {
				// Just seek to the start then.  timestamp is probably between
				// the end of the previous one and the start of this one.
				return r.SeekStart()
			} else if errors.Is(err, errTSNotFound) {
				return err
			} else {
				return fmt.Errorf("seekts: file at index %d: %w", i, err)
			}
		}

		// The search is finished, and the searched element has been found.
		// Update currentFile only, position is already set properly in
		// qLogFile.
		r.currentFile = i

		return nil
	}

	if err != nil {
		return fmt.Errorf("seekts: %w", err)
	}

	return nil
}

// SeekStart changes the current position to the end of the newest file.
// Please note that we're reading query log in the reverse order and that's why
// the log starts actually at the end of file.
//
// Returns nil if we were able to change the current position.  Returns error
// in any other cases.
func (r *qLogReader) SeekStart() error {
	if len(r.qFiles) == 0 {
		return nil
	}

	r.currentFile = len(r.qFiles) - 1
	_, err := r.qFiles[r.currentFile].SeekStart()

	return err
}

// ReadNext reads the next line (in the reverse order) from the query log
// files.  Then shifts the current position left to the next (actually prev)
// line (or the next file).
//
// Returns io.EOF if there is nothing more to read.
func (r *qLogReader) ReadNext() (string, error) {
	if len(r.qFiles) == 0 {
		return "", io.EOF
	}

	for r.currentFile >= 0 {
		q := r.qFiles[r.currentFile]
		line, err := q.ReadNext()
		if err != nil {
			// Shift to the older file.
			r.currentFile--
			if r.currentFile < 0 {
				break
			}

			q = r.qFiles[r.currentFile]

			// Set its position to the start right away.
			_, err = q.SeekStart()
			// This is unexpected, return an error right away.
			if err != nil {
				return "", err
			}
		} else {
			return line, nil
		}
	}

	// Nothing to read anymore.
	return "", io.EOF
}

// Close closes the qLogReader.
func (r *qLogReader) Close() error {
	return closeQFiles(r.qFiles)
}

// closeQFiles is a helper method to close multiple qLogFile instances.
func closeQFiles(qFiles []*qLogFile) (err error) {
	var errs []error

	for _, q := range qFiles {
		err = q.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Annotate(errors.Join(errs...), "closing qLogReader: %w")
}
