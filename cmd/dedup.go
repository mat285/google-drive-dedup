package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type data struct {
	queue chan string
	dups  chan string
	errs  chan error

	hashes map[string]string
	lock   *sync.Mutex

	wg *sync.WaitGroup
}

const (
	threads = 16
)

func dedupDrive(directory *string, delete *bool) error {
	if directory == nil || len(*directory) == 0 {
		return fmt.Errorf("Must specify --directory")
	}

	items := []string{}
	err := filepath.Walk(*directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}
		if !info.IsDir() {
			items = append(items, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	d := &data{
		queue: make(chan string, len(items)),
		dups:  make(chan string, len(items)),
		errs:  make(chan error, len(items)),

		hashes: make(map[string]string),
		lock:   new(sync.Mutex),

		wg: new(sync.WaitGroup),
	}

	for _, f := range items {
		d.queue <- f
	}
	close(d.queue)

	for i := 0; i < threads; i++ {
		d.wg.Add(1)
		go work(d)
	}
	d.wg.Wait()

	dups := printInfo(d)

	if delete == nil || !*delete {
		fmt.Println("To delete the duplicate files run with --delete=true")
		return nil
	}

	for _, d := range dups {
		fmt.Println("Deleting ", d)
		if err := os.Remove(d); err != nil {
			fmt.Println("Error deleting", d, ":", err)
		}
	}
	return nil
}

func printInfo(d *data) []string {
	errs := make([]error, len(d.errs))
	dups := make([]string, len(d.dups))

	for i := 0; i < len(d.errs); i++ {
		errs[i] = <-d.errs
	}

	for i := 0; i < len(d.dups); i++ {
		dups[i] = <-d.dups
	}

	fmt.Printf("Errors: %v\n\n\n", errs)
	fmt.Printf("Duplicates: %v\n", dups)

	return dups
}

func work(d *data) {
	defer d.wg.Done()
	for item := range d.queue {
		fmt.Println("Processing file:", item)
		f, err := os.Open(item)
		if err != nil {
			fmt.Printf("Error with file %s : %v\n", item, err)
			d.errs <- err
			f.Close()
		}
		hash, err := hashFile(f)
		if err != nil {
			fmt.Printf("Error with file %s : %v\n", item, err)
			d.errs <- err
			f.Close()
		}
		fmt.Printf("File %s hash %s\n", item, string(hash))
		dup := false
		d.lock.Lock()
		if p, has := d.hashes[string(hash)]; has && p != "" {
			dup = true
		} else {
			d.hashes[string(hash)] = item
		}
		d.lock.Unlock()
		if dup {
			d.dups <- item
		}
		f.Close()
		fmt.Println("Done processing", item)
		continue
	}
}

func hashFile(f *os.File) ([]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
