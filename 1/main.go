package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

func main() {
	var out bytes.Buffer
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(&out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func recDir(out *bytes.Buffer, path string, printFiles bool, prefix string) {
	var entries []fs.DirEntry
	rawEntries, _ := os.ReadDir(path)
	if printFiles {
		entries = rawEntries
	} else {
		for _, item := range rawEntries {
			if item.IsDir() {
				entries = append(entries, item)
			}
		}
	}

	for i, entry := range entries {
		if !entry.IsDir() && !printFiles {
			continue
		}

		out.WriteString(prefix)

		if i == len(entries)-1 {
			out.WriteString("└───")
		} else {
			out.WriteString("├───")
		}

		out.WriteString(entry.Name())

		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				out.WriteString(" (")
				sizeText := "empty"
				if info.Size() > 0 {
					sizeText = strconv.FormatInt(info.Size(), 10)
					sizeText += "b"
				}
				out.WriteString(sizeText)
				out.WriteString(")")
			}
		}

		out.WriteString("\n")
		if entry.IsDir() {
			newPrefix := prefix
			if i == len(entries)-1 {
				newPrefix += "\t"
			} else {
				newPrefix += "│\t"
			}
			recDir(out, filepath.Join(path, entry.Name()), printFiles, newPrefix)
		}
	}
}

func dirTree(out *bytes.Buffer, path string, printFiles bool) error {
	recDir(out, path, printFiles, "")
	return nil
}
