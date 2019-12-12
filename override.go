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
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var override = make(map[string][]License)

func loadOverrides() {
	filepath.Walk(".", func(name string, info os.FileInfo, err error) error {
		if filepath.Base(name) == `.git` {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(name, `.dependency_license`) {
			loadOverrideFile(name, false)
		}
		if strings.Contains(name, `.dependency_licenses`+string(os.PathSeparator)) {
			loadOverrideFile(name, true)
		}
		return nil
	})
}

func loadOverrideFile(overrideFile string, isDir bool) {
	if _, err := os.Stat(overrideFile); err != nil {
		return
	}

	f, err := os.Open(overrideFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	prefix := filepath.Dir(overrideFile)
	if isDir {
		prefix = filepath.Dir(prefix) // for files in .dependency_files directories, apply the regexes to the parent directory
	}
	if prefix == `.` {
		prefix = ``
	} else {
		prefix = regexp.QuoteMeta(prefix + string(filepath.Separator))
	}

	type licenseFilter struct {
		License License
		Regexp  *regexp.Regexp
	}

	var regexps []licenseFilter

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		line = strings.TrimSpace(line)
		if line == `` || line[0] == '#' {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			panic("Malformed line in " + overrideFile + ": " + line)
		}

		strRe, lic := strings.Join(parts[:len(parts)-1], `,`), parts[len(parts)-1]
		licParts := strings.SplitN(lic, `#`, 2)
		if len(licParts) > 1 {
			lic = licParts[0]
		}
		lic = strings.TrimSpace(lic)

		if len(strRe) > 0 && strRe[0] == '^' {
			strRe = `^` + prefix + strRe[1:]
		} else {
			strRe = `^` + prefix + ".*" + strRe
		}
		re, cmpErr := regexp.Compile(strRe)
		if cmpErr != nil {
			panic("Malformed regexp: " + strRe + "\n" + cmpErr.Error())
		}

		regexps = append(regexps, licenseFilter{License(lic), re})
	}

	err = filepath.Walk(`.`, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, filter := range regexps {
			if filter.Regexp.MatchString(path) {
				override[path] = append(override[path], filter.License)
			}
		}

		return nil
	})

	if err != nil {
		panic(`Failed when enumerating working directory: ` + err.Error())
	}
}
