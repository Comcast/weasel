/*
Copyright 2017 Comcast Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
)

// Version is the application version number for weasel
const Version = "0.0.3"

func main() {
	quiet := true
	cd := ``
	argDone := false
	nextFile := false
	logFile := ``
	nextSubdir := false
	profile := false
	subdir := ``
	printVersion := false

	for _, arg := range os.Args[1:] {
		if nextFile {
			nextFile = false
			logFile = arg
			continue
		}
		if nextSubdir {
			nextSubdir = false
			subdir = arg
			continue
		}
		if !argDone {
			if arg == `-q` {
				quiet = true
				continue
			}
			if arg == `-a` {
				quiet = false
				continue
			}
			if arg == `-f` {
				nextFile = true
				continue
			}
			if arg == `-d` {
				nextSubdir = true
				continue
			}
			if arg == `-p` {
				profile = true
				continue
			}
			if arg == `-v` {
				printVersion = true
				continue
			}
			if arg == `--` {
				argDone = true
				continue
			}
		}
		if cd == `` {
			cd = arg
			continue
		}
		fmt.Println("Unknown argument: `" + arg + "`!")
		os.Exit(1)
		return
	}
	if printVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	if profile {
		pf, err := os.Create("weasel.pprof")
		if err != nil {
			fmt.Println("Unable to start profiling: " + err.Error())
			profile = false
		} else {
			err = pprof.StartCPUProfile(pf)
			if err != nil {
				fmt.Println("Unable to start profiling: " + err.Error())
				profile = false
			}
		}
	}

	var w io.Writer
	if logFile == `` {
		w = os.Stdout
	} else {
		/* Check for directory existence. */
		logDir := filepath.Dir(logFile)
		if fi, err := os.Stat(logDir); err != nil {
			err := os.MkdirAll(logDir, 0777)
			if err != nil {
				fmt.Println("Cannot create log directory: " + err.Error())
				os.Exit(1)
				return
			}
		} else {
			if !fi.IsDir() {
				fmt.Println("Cannot create log directory, not a directory: " + logDir)
				os.Exit(1)
				return
			}
		}

		f, err := os.Create(logFile)
		if err != nil {
			fmt.Println("Cannot create log file: " + logFile)
			os.Exit(1)
			return
		}
		defer f.Close() // Won't get called on os.Exit, but that's ok, since the OS will do it for us.
		w = io.MultiWriter(os.Stdout, f)
	}

	if subdir != `` {
		var err error
		subdir, err = filepath.Abs(subdir)
		if err != nil {
			fmt.Fprintln(w, "Unable to get absolute directory for -d: "+err.Error())
			os.Exit(1)
			return
		}
	}

	if cd == `` {
		/* Find the .git directory. */
		p, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(w, "Unable to get working directory: "+err.Error())
			return
		}
		p = strings.TrimRight(p, `/`)

		patience := 10000 /* patience exists in case there are loops or other excessively long paths. */
		for p != `` && patience != 0 {
			if fi, err := os.Stat(filepath.Join(p, ".git")); err == nil && fi.IsDir() {
				cd = p
				break
			}
			p, _ = filepath.Split(p)
			p = strings.TrimRight(p, `/`)

			patience--
		}
	}
	if !quiet {
		fmt.Fprintln(w, "In directory: "+cd)
	}
	err := os.Chdir(cd)
	if err != nil {
		fmt.Fprintln(w, "Failed to enter target directory: "+err.Error()+"!")
		os.Exit(1)
		return
	}

	if subdir == `` {
		subdir = `.`
	} else {
		cur, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(w, "Failed to get working dir: "+err.Error())
			os.Exit(1)
			return
		}
		subdir, err = filepath.Rel(cur, subdir)
		if err != nil {
			fmt.Fprintln(w, "Failed to get relative subdir: "+err.Error())
			os.Exit(1)
			return
		}
	}

	loadOverrides()
	recordDocumentedLicenses()

	files := make(map[string][]License)
	var wg sync.WaitGroup
	var filesLock sync.Mutex
	err = filepath.Walk(subdir, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(name) == `.git` {
			return filepath.SkipDir
		}

		if Ignored(name) {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if (info.Mode() & os.ModeSymlink) != 0 {
			return nil
		}

		if info.Size() == 0 {
			filesLock.Lock()
			defer filesLock.Unlock()
			files[name] = append(files[name], License("Empty"))
			return nil
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			licenses, err := fileLicenses(name)
			if err != nil {
				licenses = []License{License("Error: " + err.Error() + "!")}
			}

			filesLock.Lock()
			defer filesLock.Unlock()
			files[name] = append(files[name], override[name]...)
			files[name] = append(files[name], licenses...)
			files[name] = Collide(Uniq(files[name]))
		}(name)
		return nil
	})
	wg.Wait()
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

forUnknownFiles:
	for name, licenses := range files {
		if len(licenses) == 0 {
			parts := strings.Split(name, `/`)
			for i := len(parts) - 1; i > 0; i-- {
				for _, licName := range []string{`LICENSE`, `LICENCE`, `LICENSE.md`, `LICENCE.md`, `LICENSE.txt`, `LICENCE.txt`} {
					licPath := strings.Join(parts[:i], `/`) + `/` + licName
					if len(files[licPath]) != 0 {
						for _, license := range files[licPath] {
							if license != License(`Docs`) {
								files[name] = append(files[name], License(string(license)+"~"))
							}
						}
						continue forUnknownFiles
					}
				}
			}
		}
	}

	for name, licenses := range files {
		if len(licenses) != 0 {
			if len(licenses) > 1 || (licenses[0] != License(`Apache`) && licenses[0] != License(`Docs`) && licenses[0] != License(`Empty`) && licenses[0] != License(`Ignore`)) {
				if !documented.Documents(name) {
					for i, lic := range licenses {
						if lic != License(`Apache`) && lic != License(`Docs`) && lic != License(`Empty`) && lic != License(`Ignore`) {
							licenses[i] = License(string(licenses[i]) + `!`)
						}
					}
				}
			}
		}
	}

	for name, licenses := range files {
		if len(licenses) == 0 {
			kind := filekind(name)
			if kind != `` {
				files[name] = []License{License(kind)}
			}
		}
	}

	var filenames []string
	for filename := range files {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)

	failed := false
	for _, filename := range filenames {
		lics := files[filename]
		ignore := false
		undoc := false
		var licStr string
		if len(lics) == 0 {
			licStr = "Unknown!"
			undoc = true
		} else {
			licStr = fmt.Sprint(lics[0])
			ignore = (licStr == `Ignore`)
			if len(licStr) > 0 && licStr[len(licStr)-1] == '!' {
				undoc = true
			}
			for _, lic := range lics[1:] {
				if string(lic) == `Ignore` {
					ignore = true
				}
				licStr = licStr + `, ` + fmt.Sprint(lic)
			}
		}
		if !ignore {
			errStr := ""
			if undoc {
				errStr = "Error"
				failed = true
			}
			if undoc || !quiet {
				fmt.Fprintf(w, "%-6s%40s %s\n", errStr, licStr, filename)
			}
		}
	}
	for _, extra := range documented.Extra() {
		fmt.Fprintf(w, "%-6s%40s %s\n", "Error", "Extra-License!", extra)
		failed = true
	}

	if profile {
		pprof.StopCPUProfile()
	}
	if failed {
		os.Exit(1)
	}
	os.Exit(0)
}

func fileLicenses(name string) ([]License, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	return identifyLicenses(f)
}

func identifyLicenses(in io.Reader) ([]License, error) {

	ch := make(chan string, 32)
	go func() {
		s := bufio.NewScanner(in)
		s.Split(bufio.ScanWords)
		for s.Scan() {
			s := strings.ToLower(stripPunc(s.Text()))
			if len(s) > 0 {
				ch <- s
			}
		}
		close(ch)
	}()

	licenses := newMultiMatcher(ch)
	return licenses, nil
}
