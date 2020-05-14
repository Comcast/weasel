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
	"io/ioutil"
	"os"
	"os/exec"
)

var hasGit bool

func init() {
	if _, err := exec.LookPath(`git`); err == nil {
		hasGit = true
	}
}

func Ignored(f string) bool {
	if hasGit {
		if tmpGitDir != "" {
			_, err := exec.Command(`git`, `--git-dir=`+tmpGitDir+"/.git", `check-ignore`, `-q`, f).CombinedOutput()
			return err == nil
		} else {
			_, err := exec.Command(`git`, `check-ignore`, `-q`, f).CombinedOutput()
			return err == nil
		}
	}
	return false
}

var tmpGitDir string

func initGit() {
	if hasGit {
		if _, err := os.Stat(`.git`); os.IsNotExist(err) {
			dir, err := ioutil.TempDir("", "weasel-git-")
			if err != nil {
				return
			}
			tmpGitDir = dir

			cmd := exec.Command(`git`, `init`)
			cmd.Dir = tmpGitDir
			_, err = cmd.CombinedOutput()
			if err != nil {
				cleanupGit()
				return
			}
		}
	}
}

func cleanupGit() {
	if tmpGitDir != "" {
		os.RemoveAll(tmpGitDir)
	}
}
