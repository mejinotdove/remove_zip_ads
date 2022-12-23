package main

import (
	"archive/zip"
	"fmt"
	"github.com/EDDYCJY/gsema"
	"github.com/corona10/goimagehash"
	"github.com/google/uuid"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
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
				h, err := goimagehash.PerceptionHash(*i)
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

var sema = gsema.NewSemaphore(runtime.NumCPU())

func getImg(r io.Reader, fn string) (*image.Image, error) {
	defer sema.Done()
	sema.Add(1)

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

func deleteAndBackup(zipPath string, f *zip.File, sn string, distance int, mu *sync.Mutex, filesToIgnore *[]string, dfc chan<- *DeletedFile) {
	fmt.Printf("%s is very similary to %s, distance: %d\n", f.Name, sn, distance)
	mu.Lock()
	defer mu.Unlock()
	*filesToIgnore = append(*filesToIgnore, f.Name)

	wd, err := os.Getwd()
	if err != nil {
		log.Print(err)
		return
	}

	dn := uuid.NewString()
	df, err := os.Create(filepath.Join(wd, "deleted_files", dn+filepath.Ext(f.Name)))
	if err != nil {
		log.Print(err)
		return
	}

	fc, err := f.Open()
	if err != nil {
		log.Print(err)
		return
	}

	_, err = io.Copy(df, fc)
	if err != nil {
		log.Print(err)
		return
	}

	dfc <- &DeletedFile{
		CurrentFileName:  dn,
		OriginalFileName: f.Name,
		ZipFilePath:      zipPath,
	}
}

func checkAndProcess(wg *sync.WaitGroup, zipPath string, f *zip.File, hashs map[string]*goimagehash.ImageHash, mu *sync.Mutex, filesToIgnore *[]string, dfc chan<- *DeletedFile) {
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
	h, err := goimagehash.PerceptionHash(*i)
	if err != nil {
		log.Print(err)
		return
	}

	for sn, sh := range hashs {
		if distance, _ := h.Distance(sh); distance < 10 {
			deleteAndBackup(zipPath, f, sn, distance, mu, filesToIgnore, dfc)
			return
		}
	}
}

func process(path string, totalFiles int, idx int, hashs map[string]*goimagehash.ImageHash, recordDeletedFilesChan chan *DeletedFile) error {
	fmt.Printf("processing %s\t %d of %d\n", filepath.Base(path), idx+1, totalFiles)
	z, err := zip.OpenReader(path)
	if err != nil {
		log.Fatal(err)
	}
	defer z.Close()

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	var filesToIgnore []string
	for _, f := range z.File {
		wg.Add(1)
		go checkAndProcess(&wg, path, f, hashs, &mu, &filesToIgnore, recordDeletedFilesChan)
	}

	wg.Wait()
	if len(filesToIgnore) > 0 {
		tmpZipFilePath := path + ".tmp"
		tmpZipFile, err := os.Create(tmpZipFilePath)
		if err != nil {
			return err
		}
		tmpZipFileWriter := zip.NewWriter(tmpZipFile)
	PROCESS:
		for _, f := range z.File {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			for _, ignoreFile := range filesToIgnore {
				if ignoreFile == f.Name {
					continue PROCESS
				}
			}

			w, err := tmpZipFileWriter.Create(f.Name)
			if err != nil {
				return err
			}
			_, err = io.Copy(w, rc)
			if err != nil {
				return err
			}
		}
		tmpZipFileWriter.Close()
		os.RemoveAll(path)
		os.Rename(tmpZipFilePath, path)
	}
	return nil
}

func start2(targetPath string) {
	recordDeletedFilesChan := make(chan *DeletedFile)
	closedChan := make(chan struct{}, 1)
	go recordDeletedFiles(recordDeletedFilesChan, closedChan)

	hashs, _ := readAdSamplesHash()

	fmt.Printf("******************************************\n")

	var targetFilesPath []string
	err := filepath.Walk(targetPath, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() || filepath.Ext(strings.ToLower(info.Name())) != ".zip" {
			return nil
		}

		targetFilesPath = append(targetFilesPath, path)
		return nil
	})

	totalFiles := len(targetFilesPath)
	for idx, path := range targetFilesPath {
		process(path, totalFiles, idx, hashs, recordDeletedFilesChan)
	}
	if err != nil {
		log.Fatal(err)
	}

	close(recordDeletedFilesChan)
	<-closedChan
}
