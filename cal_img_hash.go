package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"github.com/corona10/goimagehash"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func readAdSamplesHash() (map[string]*goimagehash.ImageHash, error) {
	fmt.Printf("Start read ad samples...\n")
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	samplePath := wd + "/ad_samples/"

	wg := sync.WaitGroup{}

	mu := sync.Mutex{}
	hashs := make(map[string]*goimagehash.ImageHash)

	fmt.Printf("sample path: %s\n", samplePath)
	err = filepath.Walk(wd+"/ad_samples/", func(path string, info fs.FileInfo, err error) error {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if info.IsDir() {
				return
			}

			if b, err := os.Open(path); err != nil {
				panic(err)
			} else {
				defer b.Close()
				i, err := getImg(b, path)
				if err != nil {
					panic(err)
				}
				h, err := goimagehash.PerceptionHash(i)
				if err != nil {
					panic(err)
				}

				fmt.Printf("%s hash: %d\n", info.Name(), h.ToString())
				mu.Lock()
				defer mu.Unlock()
				hashs[info.Name()] = h
			}
		}()
		return nil
	})

	if err != nil {
		return nil, err
	}

	wg.Wait()
	fmt.Printf("Read ad samples completed\n")
	return hashs, nil
}

func getImg(r io.Reader, fn string) (image.Image, error) {
	switch filepath.Ext(strings.ToLower(fn)) {
	case ".jpg", ".jpeg":
		return jpeg.Decode(r)
	case ".png":
		return png.Decode(r)
	case ".gif":
		return gif.Decode(r)
	default:
		return nil, errors.New(fn + " is not valid img file")
	}
}

func start2(targetPath string) {
	hashs, _ := readAdSamplesHash()

	fmt.Printf("******************************************\n")

	cnt := 0
	err := filepath.Walk(targetPath, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() || filepath.Ext(strings.ToLower(info.Name())) != ".zip" || (info.Size()/1024/1024) > 600 {
			return nil
		}

		fmt.Printf("processing %s\n", info.Name())
		z, err := zip.OpenReader(path)
		if err != nil {
			log.Fatal(err)
		}
		defer z.Close()

		wg := sync.WaitGroup{}
		mu := sync.Mutex{}
		var resultStr []string
		for _, fp := range z.File {
			f := fp
			wg.Add(1)
			go func() {
				defer wg.Done()
				fc, err := f.Open()
				if err != nil {
					log.Print(err)
					return
				}
				defer fc.Close()
				i, err := getImg(fc, f.Name)
				if err != nil {
					log.Print(err)
					return
				}
				h, err := goimagehash.PerceptionHash(i)
				if err != nil {
					log.Print(err)
					return
				}
				for sn, sh := range hashs {
					if distance, _ := h.Distance(sh); distance < 10 {
						mu.Lock()
						resultStr = append(resultStr, fmt.Sprintf("%s is very similary to %s, distance: %d\n", f.Name, sn, distance))
						mu.Unlock()
					}
				}
			}()
		}

		wg.Wait()
		cnt++
		fmt.Printf("==== %s \t%d ====\n", info.Name(), cnt)
		for _, s := range resultStr {
			fmt.Printf(s)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
