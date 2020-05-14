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

SPDX-License-Identifier: Apache-2.0
*/

package main

//go:generate bash make_licenses.sh

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/google/licenseclassifier"
	"github.com/google/licenseclassifier/stringclassifier"
)

// Version is the application version number for weasel
const Version = "0.0.4"

func main() {
	var all bool
	flag.BoolVar(&all, "a", false, "Print all files and their licenses, not just problematic files.")
	var subdir string
	flag.StringVar(&subdir, "d", "", "Only run on files in the specified subdirectory")
	var logFile string
	flag.StringVar(&logFile, "f", "", "Send output to this file (in addition to stdout)")
	var profile bool
	flag.BoolVar(&profile, "p", false, "Collect and output profiling statistics.")
	var printVersion bool
	flag.BoolVar(&printVersion, "v", false, "Print version and exit.")
	_ = flag.Bool("q", true, "Only print problematic files. DEPRECATED: as of v0.0.4 this flag is deprecated and does nothing - just use -a or its absence.")
	flag.Parse()
	quiet := !all

	args := flag.Args()
	var cd string
	if len(args) > 0 {
		cd = args[len(args)-1]
	}

	if printVersion {
		fmt.Println(Version)
		exit(0)
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
				exit(1)
				return
			}
		} else {
			if !fi.IsDir() {
				fmt.Println("Cannot create log directory, not a directory: " + logDir)
				exit(1)
				return
			}
		}

		f, err := os.Create(logFile)
		if err != nil {
			fmt.Println("Cannot create log file: " + logFile)
			exit(1)
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
			exit(1)
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
	if cd != `` {
		err := os.Chdir(cd)
		if err != nil {
			fmt.Fprintln(w, "Failed to enter target directory: "+err.Error()+"!")
			exit(1)
			return
		}
	}
	initGit()

	if subdir == `` {
		subdir = `.`
	} else {
		cur, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(w, "Failed to get working dir: "+err.Error())
			exit(1)
			return
		}
		subdir, err = filepath.Rel(cur, subdir)
		if err != nil {
			fmt.Fprintln(w, "Failed to get relative subdir: "+err.Error())
			exit(1)
			return
		}
	}

	loadOverrides()
	recordDocumentedLicenses()

	files := make(map[string][]License)
	var wg sync.WaitGroup
	var filesLock sync.Mutex
	throttle := make(chan struct{}, 32)
	err := filepath.Walk(subdir, func(name string, info os.FileInfo, err error) error {
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
			throttle <- struct{}{}
			defer func() { <-throttle }()
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
			if len(licenses) > 1 || (licenses[0] != License(`Apache-2.0`) && licenses[0] != License(`Docs`) && licenses[0] != License(`Empty`) && licenses[0] != License(`Ignore`)) {
				if !documented.Documents(name) {
					for i, lic := range licenses {
						if lic != License(`Apache-2.0`) && lic != License(`Docs`) && lic != License(`Empty`) && lic != License(`Ignore`) {
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
		exit(1)
	}
	exit(0)
}

func fileLicenses(name string) ([]License, error) {
	spdx, err := spdxLicenses(name)
	if err != nil {
		return nil, err
	}
	if len(spdx) > 0 {
		return spdx, nil // If they provided an explicit SPDX id, just use that.
	}

	fi, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	if fi.Size() > 2*1024*1024 {
		return nil, nil
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return identifyLicenses(f)
}

func spdxLicenses(name string) ([]License, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	const maxBuffer = 2 * 1024 // Only check the first and last 10k of the file, for performance.

	if fi.Size() < maxBuffer {
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("Unable to read all of file %s: %v", name, err)
		}
		return spdxLicenseSearch(b), nil
	}

	b := make([]byte, maxBuffer)
	n, err := f.Read(b)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("Unable to read top of %s: %v", name, err)
	}
	topLicenses := spdxLicenseSearch(b[:n])

	tail := fi.Size() - maxBuffer
	if tail < maxBuffer {
		tail = maxBuffer
	}
	n, err = f.ReadAt(b, tail)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("Unable to read tail of %s: %v", name, err)
	}
	tailLicenses := spdxLicenseSearch(b[:n])
	return append(topLicenses, tailLicenses...), nil
}

func spdxLicenseSearch(b []byte) []License {
	spdxShort := []byte("SPDX-License-Identifier:")

	var licenses []License
	lines := bytes.Split(b, []byte("\n"))
forLines:
	for _, line := range lines {
		idx := bytes.Index(line, spdxShort)
		if idx >= 0 {
			prefix := line[:idx]
			prefixAlpha := 0
			for _, c := range string(prefix) {
				if unicode.IsLetter(c) {
					prefixAlpha++
					if prefixAlpha > 5 {
						continue forLines
					}
				}
			}

			suffixIdx := idx + len(spdxShort)
			if suffixIdx >= len(line) {
				continue forLines
			}
			suffix := bytes.Trim(line[suffixIdx:], ` `)
			licenses = append(licenses, License(suffix))
		}
	}
	return licenses
}

var classifier *licenseclassifier.License

func init() {
	var err error
	classifier, err = licenseclassifier.New(0.8, licenseclassifier.ArchiveBytes(LicenseDBContents))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize classifier: %v\n", err)
		exit(-1)
	}
}

func identifyLicenses(in io.Reader) ([]License, error) {
	var licenses Licenses
	b, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, fmt.Errorf("Unable to read all of file: %v", err)
	}

	var matches stringclassifier.Matches
	if err == nil {
		matches = classifier.MultipleMatch(string(b), true)
	} else {
		return nil, fmt.Errorf("Cannot create classifier: %v\n", err)
	}

	for _, match := range matches {
		if match != nil {
			licenses = append(licenses, License(match.Name))
		}
	}
	return licenses, nil
}

func exit(code int) {
	cleanupGit()
	os.Exit(code)
}
