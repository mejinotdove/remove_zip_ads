package main

import (
	"encoding/csv"
	"log"
	"os"
	"path/filepath"
)

type DeletedFile struct {
	CurrentFileName  string // 当前文件名, MD5
	OriginalFileName string // 原始文件名
	ZipFilePath      string // 所属ZIP文件路径
}

// 记录被删除
func recordDeletedFiles(df <-chan *DeletedFile, done chan<- struct{}) {
	defer func() {
		done <- struct{}{}
	}()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	cf, err := os.OpenFile(filepath.Join(wd, "deleted_files", "files.csv"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	w := csv.NewWriter(cf)
	for {
		select {
		case d, ok := <-df:
			if ok == false {
				w.Flush()
				return
			}

			line := []string{d.CurrentFileName, d.OriginalFileName, d.ZipFilePath}
			err := w.Write(line)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

//
//func readAdSamplesCRC32() (*gset.Set, error) {
//	var adListCRC32 = gset.New(true)
//	fmt.Printf("Start read ad samples...\n")
//	wd, err := os.Getwd()
//	if err != nil {
//		return nil, err
//	}
//	samplePath := wd + "/ad_samples/"
//	fmt.Printf("sample path: %s\n", samplePath)
//	err = filepath.Walk(wd+"/ad_samples/", func(path string, info fs.FileInfo, err error) error {
//		if info.IsDir() {
//			return nil
//		}
//
//		if b, err := os.ReadFile(path); err != nil {
//			return err
//		} else {
//			crc := crc32.ChecksumIEEE(b)
//			fmt.Printf("%s crc32: %d\n", path, crc)
//			adListCRC32.Add(crc)
//		}
//		return nil
//	})
//
//	if err != nil {
//		return nil, err
//	}
//
//	fmt.Printf("Read ad samples completed\n")
//	return adListCRC32, nil
//}
//
//func checkAndProcess(wg *sync.WaitGroup, adSet *gset.Set, path string, dfc chan<- *DeletedFile) error {
//	defer func() {
//		wg.Done()
//	}()
//
//	originalZip, err := zip.OpenReader(path)
//	if err != nil {
//		return err
//	}
//	defer originalZip.Close()
//
//	originalCRCs := gset.New(true)
//	for _, f := range originalZip.File {
//		originalCRCs.Add(f.CRC32)
//	}
//
//	intersect := adSet.Intersect(originalCRCs)
//	if intersect.Size() == 0 {
//		fmt.Printf("%s no ad, done\n", path)
//		return nil
//	}
//
//	fmt.Printf("%s has %d ads, begin to delete ads\n", path, intersect.Size())
//	tmpZipFilePath := path + ".tmp"
//	tmpZipFile, err := os.Create(tmpZipFilePath)
//	if err != nil {
//		return err
//	}
//	tmpZipFileWriter := zip.NewWriter(tmpZipFile)
//	for _, f := range originalZip.File {
//		rc, err := f.Open()
//		if err != nil {
//			return err
//		}
//
//		if adSet.Contains(f.CRC32) {
//			wd, err := os.Getwd()
//			if err != nil {
//				return err
//			}
//
//			dn := uuid.NewString()
//			df, err := os.Create(filepath.Join(wd, "deleted_files", dn+filepath.Ext(f.Name)))
//			if err != nil {
//				return err
//			}
//
//			_, err = io.Copy(df, rc)
//			if err != nil {
//				return err
//			}
//
//			dfc <- &DeletedFile{
//				CurrentFileName:  dn,
//				OriginalFileName: f.Name,
//				ZipFilePath:      path,
//			}
//			continue
//		}
//
//		w, err := tmpZipFileWriter.Create(f.Name)
//		if err != nil {
//			return err
//		}
//		_, err = io.Copy(w, rc)
//		if err != nil {
//			return err
//		}
//	}
//	tmpZipFileWriter.Close()
//	os.RemoveAll(path)
//	os.Rename(tmpZipFilePath, path)
//	fmt.Printf("process %s success\n", path)
//	return nil
//}
//
//func start1(targetPath string) {
//	recordDeletedFilesChan := make(chan *DeletedFile)
//	closedChan := make(chan struct{}, 1)
//
//	wg := sync.WaitGroup{}
//
//	go recordDeletedFiles(recordDeletedFilesChan, closedChan)
//
//	ads, err := readAdSamplesCRC32()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	filepath.Walk(targetPath, func(path string, info fs.FileInfo, err error) error {
//		if info.IsDir() || filepath.Ext(path) != ".zip" {
//			return nil
//		}
//
//		wg.Add(1)
//		go func() {
//			if err := checkAndProcess(&wg, ads, path, recordDeletedFilesChan); err != nil {
//				log.Println(err)
//			}
//		}()
//		return nil
//	})
//
//	wg.Wait()
//	close(recordDeletedFilesChan)
//	<-closedChan
//}

func main() {
	var targetPath string
	if len(os.Args) < 2 {
		t, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		targetPath = t
	} else {
		targetPath = os.Args[1]
	}

	start2(targetPath)
}
