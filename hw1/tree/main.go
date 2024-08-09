package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, testdata bool) error {
	writeTree(out, path, testdata, "")
	return nil
}

func onlyDirs(files []os.DirEntry) []os.DirEntry {
	dirs := make([]os.DirEntry, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file)
		}
	}
	return dirs
}

func writeTree(out io.Writer, path string, testdata bool, prefix string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	files, err := f.ReadDir(0)
	if err != nil {
		return err
	}
	if !testdata {
		files = onlyDirs(files)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for idx, file := range files {
		str := ""
		isLast := idx+1 == len(files)
		if !isLast {
			str = "├───"
		} else {
			str = "└───"
		}
		if file.IsDir() {
			fmt.Fprintf(out, prefix+str+"%s\n", file.Name())
			if isLast {
				writeTree(out, path+"/"+file.Name(), testdata, prefix+"\t")
			} else {
				writeTree(out, path+"/"+file.Name(), testdata, prefix+"│\t")
			}
			continue
		}
		count, err := os.ReadFile(path + "/" + file.Name())
		if err != nil {
			return err
		}
		bytes := strconv.Itoa(len(count)) + "b"
		if len(count) == 0 {
			bytes = "empty"
		}
		fmt.Fprintf(out, prefix+str+"%s (%v)\n", file.Name(), bytes)
	}
	return nil
}
