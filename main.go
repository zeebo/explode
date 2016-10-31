package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
)

type stackError struct {
	err   error
	stack string
}

func (s stackError) Error() string {
	return fmt.Sprintf("%v\n%s", s.err, s.stack)
}

func wrap(err error) error {
	if err == nil {
		return nil
	}
	var buf [4096]byte
	return stackError{
		err:   err,
		stack: string(buf[:runtime.Stack(buf[:], true)]),
	}
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, wrap(fmt.Errorf(format, args...)))
	os.Exit(2)
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s input output\n", os.Args[0])
		os.Exit(1)
	}
	if err := run(context.Background(), args[0], args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
