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

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/licenseclassifier/serializer"
)

func main() {
	if len(os.Args) != 3 {
		bail()
	}
	dir := os.Args[1]
	licensedb := os.Args[2]
	if dir == `` || licensedb == `` {
		bail()
	}

	out, err := os.Create(licensedb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create %s: %v\n", licensedb, err)
		os.Exit(1)
	}
	defer out.Close()

	var licenses []string
	fis, err := ioutil.ReadDir(dir)
	for _, fi := range fis {
		absPath, err := filepath.Abs(filepath.Join(dir, fi.Name()))
		if err != nil {
			continue
		}
		licenses = append(licenses, absPath)
	}

	err = serializer.ArchiveLicenses(licenses, out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to serialize licenses to %s: %v\n", licensedb, err)
		os.Exit(1)
	}
}

func bail() {
	fmt.Fprintf(os.Stderr, "Usage: %s <dir> <licensedb>\n")
	os.Exit(1)
}
