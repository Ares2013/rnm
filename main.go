package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sync"

	"github.com/logrusorgru/aurora/v3"
	"github.com/mattn/go-zglob"
)

func main() {
	err := run()
	if err != nil {
		printError(err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args[1:]) != 2 {
		return errors.New("Usage: rnm <from> <to>")
	}

	r, err := newRenamer(os.Args[1], os.Args[2])
	if err != nil {
		return err
	}

	ss, err := zglob.Glob("**/*")
	if err != nil {
		return err
	}

	g := &sync.WaitGroup{}
	ec := make(chan error, 1024)

	for _, s := range ss {
		g.Add(1)
		go func(s string) {
			defer g.Done()

			ok, err := validateFilename(s)
			if err != nil {
				ec <- err
			}

			if !ok {
				return
			}

			err = renameFile(r, s)
			if err != nil {
				ec <- err
			}
		}(s)
	}

	go func() {
		g.Wait()

		close(ec)
	}()

	ok := true

	for err := range ec {
		ok = false

		printError(err)
	}

	if !ok {
		return errors.New("failed to rename some identifiers")
	}

	return nil
}

func validateFilename(s string) (bool, error) {
	ok, err := regexp.MatchString("(^|/)\\.", s)
	if err != nil {
		return false, err
	}

	return !ok, nil
}

func renameFile(r *renamer, path string) error {
	p := r.Rename(path)

	if p != path {
		err := os.Rename(path, p)
		if err != nil {
			return err
		}
	}

	i, err := os.Lstat(p)
	if err != nil {
		return err
	} else if i.IsDir() {
		return nil
	}

	bs, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(p, []byte(r.Rename(string(bs))), i.Mode())
}

func printError(err error) {
	fmt.Fprintln(os.Stderr, aurora.Red(err))
}
