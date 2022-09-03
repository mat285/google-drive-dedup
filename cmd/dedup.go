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
	stop  chan interface{}

	hashes map[string]string
	lock   *sync.Mutex

	wg *sync.WaitGroup
}

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
		stop:  make(chan interface{}),

		hashes: make(map[string]string),
		lock:   new(sync.Mutex),

		wg: new(sync.WaitGroup),
	}

	for i := 0; i < 16; i++ {
		d.wg.Add(1)
		go work(d)
	}
	d.wg.Wait()

	dups := printInfo(d)

	if delete == nil || !*delete {
		fmt.Println("To delete the duplicate files run with --delete=true")
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
	for {
		if len(d.queue) == 0 {
			close(d.stop)
			return
		}
		select {
		case item := <-d.queue:
			if len(d.queue) == 0 {
				close(d.stop)
				return
			}
			f, err := os.Open(item)
			if err != nil {
				d.errs <- err
				f.Close()
			}
			hash, err := hashFile(f)
			if err != nil {
				d.errs <- err
				f.Close()
			}
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
			continue
		case <-d.stop:
			return
		}
	}
}

func hashFile(f *os.File) ([]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}