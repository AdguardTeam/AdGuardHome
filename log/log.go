// Wrapper for standard library log, with the only difference is that it has extra function Tracef() and optional verbose flag to enable output from that.
package log

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
)

var Verbose = false

// Print calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Print.
func Print(v ...interface{}) {
	log.Print(v...)
}

// Printf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Println.
func Println(v ...interface{}) {
	log.Println(v...)
}

// Fatal is equivalent to Print() followed by a call to os.Exit(1).
func Fatal(v ...interface{}) {
	log.Fatal(v...)
}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func Tracef(format string, v ...interface{}) {
	if Verbose {
		pc := make([]uintptr, 10) // at least 1 entry needed
		runtime.Callers(2, pc)
		f := runtime.FuncForPC(pc[0])
		var buf strings.Builder
		buf.WriteString(fmt.Sprintf("%s(): ", path.Base(f.Name())))
		text := fmt.Sprintf(format, v...)
		buf.WriteString(text)
		if len(text) == 0 || text[len(text)-1] != '\n' {
			buf.WriteRune('\n')
		}
		fmt.Fprint(os.Stderr, buf.String())
	}
}
