package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var (
	Info = Teal
	Warn = Yellow
	Fata = Red
)

var (
	Black   = Color("\033[1;30m%s\033[0m")
	Red     = Color("\033[1;31m%s\033[0m")
	Green   = Color("\033[1;32m%s\033[0m")
	Yellow  = Color("\033[1;33m%s\033[0m")
	Purple  = Color("\033[1;34m%s\033[0m")
	Magenta = Color("\033[1;35m%s\033[0m")
	Teal    = Color("\033[1;36m%s\033[0m")
	White   = Color("\033[1;37m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

var (
	searchResult []string
)

func main() {

	fmt.Println()
	fmt.Println("LevelDB Dumper 2.0.0.0")
	fmt.Println()
	fmt.Println("Author: Matt Dawson")
	fmt.Println()

	getArgs := func() (string, bool, string) {
		dbPath := ""
		quiet := false
		csvPath := ""

		for i := 1; i < len(os.Args); i++ {
			if os.Args[i] == "-d" && i+1 < len(os.Args) {
				path, err := filepath.Abs(os.Args[i+1])
				if err != nil {
					fmt.Println(Fata("Unable to get absolute path of ", path))
				}
				dbPath = path
			}
			if os.Args[i] == "-q" {
				quiet = true
			}
			if os.Args[i] == "--csv" && i+1 < len(os.Args) {
				csvPath = os.Args[i+1]
			}
		}
		return dbPath, quiet, csvPath
	}

	printUsage := func() {
		fmt.Println("        d               Directory to recursively process. This is required.")
		fmt.Println("        q               Don't output all key/value pairs to console. Default will output all key/value pairs")
		fmt.Println("        csv             Directory to save CSV formatted results to. Be sure to include the full path in double quotes")
		fmt.Println()
		fmt.Println("Examples: LevelDBParser.exe -f \"C:\\Temp\\leveldb\"")
		fmt.Println("          LevelDBParser.exe -f \"C:\\Temp\\leveldb\" --csv \"C:\\Temp\" -q")
		fmt.Println()
		fmt.Println("          Short options (single letter) are prefixed with a single dash. Long commands are prefixed with two dashes")
		fmt.Println()
	}

	fileExists := func(path string) (bool, error) {
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, err
	}

	rootPath, quiet, csvPath := getArgs()

	if rootPath == "" {
		printUsage()
		fmt.Println(Fata("Missing -d argument"))
		fmt.Println()
		return
	}

	fmt.Println("Command Line:", strings.Join(os.Args[1:], " "))
	fmt.Println()

	dbPresent, _ := fileExists(rootPath)

	if !dbPresent {
		fmt.Println(Fata("The DB path ", rootPath, " doesn't exist"))
		printUsage()
		return
	}

	testFile, err := os.Open(rootPath)
	if err != nil {
		fmt.Println(Warn("Unable to open ", rootPath, " - make sure you haven't escaped the path with \\\""))
		return
	}
	defer testFile.Close()

	start := time.Now()
	err = filepath.Walk(rootPath, findFile)
	if err != nil {
		return
	}
	elapsed := time.Now().Sub(start)
	if len(searchResult) > 0 {
		fmt.Println(Warn(len(searchResult), " LevelDB databases found"))
		fmt.Println(Info("Searching for LevelDB databases from ", rootPath, " took ", elapsed))
		fmt.Println()
		for _, v := range searchResult {
			openDb(v, quiet, csvPath)
		}
	} else {
		fmt.Println(Fata("0 LevelDB databases found"))
		fmt.Println()
	}
}

func findFile(path string, fileInfo os.FileInfo, err error) error {
	if err != nil {
		fmt.Println(Warn("Access denied for ", path))
		return nil
	}

	absolute, err := filepath.Abs(path)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	if fileInfo.IsDir() {
		files, err := filepath.Glob(filepath.Join(absolute, "CURRENT"))
		checkError(err)
		if len(files) > 0 {
			files, err := filepath.Glob(filepath.Join(absolute, "MANIFEST-*"))
			checkError(err)
			if len(files) > 0 {
				searchResult = append(searchResult, absolute)
			}
		}
		return nil
	}

	return nil
}

func openDb(dbPath string, quiet bool, csvPath string) {

	fmt.Println(Info("Opening DB at ", dbPath))

	options := &opt.Options{
		ErrorIfMissing: true,
	}

	start := time.Now()

	db, err := leveldb.OpenFile(dbPath, options)

	if err != nil {
		fmt.Println(Fata("Could not open DB at ", dbPath))
		fmt.Println()
		return
	}
	fmt.Println()

	defer db.Close()

	iter := db.NewIterator(nil, nil)

	if !quiet {
		fmt.Println(Info(fmt.Sprintf("%-56vValue:", "Key:")))
	}

	var data = [][]string{}

	for iter.Next() {
		key := iter.Key()
		keyName := string(key[:])

		byteValue, err := db.Get([]byte(key), nil)
		if err != nil {
			fmt.Println("Error reading Key: " + keyName)
			return
		}
		value := string(byteValue)

		escapedKey := removeControlChars(keyName)
		escapedValue := removeControlChars(value)

		if !quiet {
			if len(escapedValue) > 80 {
				fmt.Printf("%-64v | "+escapedValue[:80]+"...\n", Warn(escapedKey))
			} else {
				fmt.Printf("%-64v | "+escapedValue+"\n", Warn(escapedKey))
			}
		}

		data = append(data, []string{keyName, value})
	}

	if csvPath != "" {
		if len(data) > 0 {
			timeNow := time.Now()
			year, month, day := timeNow.Date()
			escapedPath := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(dbPath, "/", "_"), "\\", "_"), ":", "")
			csvFileName := fmt.Sprintf("%v%v%v%v%v%v_%v_LevelDBDumper.csv", year, int(month), day, timeNow.Hour(), timeNow.Minute(), timeNow.Second(), escapedPath)
			file, err := os.Create(filepath.Join(csvPath, csvFileName))
			checkError(err)
			defer file.Close()

			csvWriter := csv.NewWriter(file)
			csvWriter.Write([]string{"Key", "Value"})

			for _, value := range data {
				err := csvWriter.Write(value)
				checkError(err)
				csvWriter.Flush()
			}
		} else {
			fmt.Println(Warn("Parsed database at ", dbPath, " but no key/value pairs were found"))
		}
	}

	iter.Release()
	err = iter.Error()
	checkError(err)
	if !quiet {
		fmt.Println()
	}

	elapsed := time.Now().Sub(start)
	fmt.Println(Info("Dumping LevelDB database at ", dbPath, " took ", elapsed))
	fmt.Println()
}

func removeControlChars(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsGraphic(r) {
			return r
		}
		return -1
	}, str)
}

func checkError(err error) {
	if err != nil {
		fmt.Println(Fata(err))
	}
}