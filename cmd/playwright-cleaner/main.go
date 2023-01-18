package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	maxResourceFileSize = flag.Int64("maxresource", 1000000 /* 1 MB */, "Maximum size for a resource file")
	maxTraceLineLength  = flag.Int("maxline", 1000000 /* 1 MB */, "Maximum size for a trace file line")
)

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Printf("Usage: %s [options] ./path/to/playwright-report", os.Args[0])
		flag.Usage()
		os.Exit(1)
	}

	dataDir := os.Args[1] + "/data"
	zipFiles, err := filepath.Glob(dataDir + "/*.zip")
	if err != nil {
		log.Fatalf("Failed to read trace files from data directory %s: %v", dataDir, err)
	}

	var errs []error
	for _, zipFile := range zipFiles {
		err := cleanZipfile(zipFile)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to process trace file %s: %v", zipFile, err))
		}
	}

	for _, err := range errs {
		log.Print(err)
	}

	if len(errs) > 0 {
		log.Fatalf("%d errors occurred cleaning %d trace files", len(errs), len(zipFiles))
	}

	log.Printf("%d trace files processed successfully", len(zipFiles))
}

func cleanZipfile(inputFileName string) error {
	reader, err := zip.OpenReader(inputFileName)
	if err != nil {
		return err
	}
	defer reader.Close()

	outputFileName := inputFileName + ".new"
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(outputFile)

	writer.SetComment(reader.Comment)

	for _, f := range reader.File {
		err := handleFile(f, writer)
		if err != nil {
			writer.Close()
			outputFile.Close()
			os.Remove(outputFileName)
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}
	if err := outputFile.Close(); err != nil {
		return err
	}

	return os.Rename(outputFileName, inputFileName)
}

func handleFile(f *zip.File, writer *zip.Writer) error {
	log.Printf("file: name %q, size %v", f.Name, f.FileInfo().Size())

	if f.Name == "trace.trace" {
		fileReader, err := f.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		fileWriter, err := writer.Create(f.Name)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(fileReader)
		scanner.Buffer(make([]byte, 64), 1*1024*1024*1024 /* 1 GB max line size, only allocated when necessary */)
		lineNum := 1
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > *maxTraceLineLength {
				line, err = shortenTraceLine(line)
				if err != nil {
					log.Printf("Failed to shorten trace line %d: %v", lineNum, err)
					// shortenTraceLine guarantees that the returned line can be used anyway
				}
			}
			if _, err := fileWriter.Write(line); err != nil {
				return err
			}
			if _, err := fileWriter.Write([]byte{'\n'}); err != nil {
				return err
			}
			lineNum += 1
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		return nil
	}

	if strings.HasPrefix(f.Name, "resources/") && f.FileInfo().Size() > *maxResourceFileSize {
		// Skip resource file, it is too large
		return nil
	}

	// Copy file as-is
	writer.Copy(f)
	return nil
}

func shortenTraceLine(line []byte) ([]byte, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(line, &obj); err != nil {
		return line, err
	}

	if metadata, ok := traverseObject(obj, "metadata"); ok {
		if params, ok := traverseObject(metadata, "params"); ok {
			if arg, ok := traverseObject(params, "arg"); ok {
				if value, ok := traverseObject(arg, "value"); ok {
					if s, ok := value["s"]; ok {
						s := s.(string)
						value["s"] = s[0:40] + "[...snip...]" + s[len(s)-40:]
					} else {
						arg["value"] = map[string]interface{}{}
					}
				}
			}
		}
	}

	newLine, err := json.Marshal(obj)
	if err != nil {
		return line, err
	}

	if len(newLine) > *maxTraceLineLength {
		return newLine, fmt.Errorf("line wasn't shorter - need to implement this type of line?")
	}
	return newLine, nil
}

func traverseObject(obj map[string]interface{}, key string) (map[string]interface{}, bool) {
	if child, ok := obj[key]; ok {
		res, ok := child.(map[string]interface{})
		return res, ok
	} else {
		return nil, false
	}
}
