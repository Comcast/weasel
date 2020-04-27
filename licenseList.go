/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.

SPDX-License-Identifier: Apache-2.0

SPDX-License-Identifier: Apache-2.0
*/
package main

import "sort"

type License string

type Licenses []License

func (lics Licenses) Len() int {
	return len(lics)
}
func (lics Licenses) Swap(i, j int) {
	lics[i], lics[j] = lics[j], lics[i]
}
func (lics Licenses) Less(i, j int) bool {
	return lics[i] < lics[j]
}

func Uniq(lics []License) []License {
	if len(lics) == 0 {
		return nil
	}
	sortLics := make(Licenses, len(lics))
	copy(sortLics, lics)
	sort.Sort(sortLics)
	var uniqLics []License
	for _, lic := range sortLics {
		if len(uniqLics) == 0 || lic != uniqLics[len(uniqLics)-1] {
			uniqLics = append(uniqLics, lic)
		}
	}
	return uniqLics
}

func Remove(lics []License, rmLic License) []License {
	var rmLics []License
	for _, lic := range lics {
		if lic != rmLic {
			rmLics = append(rmLics, lic)
		}
	}
	return rmLics
}

func Has(lics []License, lic License) bool {
	for _, l := range lics {
		if l == lic {
			return true
		}
	}
	return false
}

func Collide(lics []License) []License {
	var toRm []License
	for _, lic := range lics {
		if string(lic)[0] == '!' {
			toRm = append(toRm, lic)
			toRm = append(toRm, License(string(lic)[1:]))
		}
	}
	newLics := lics
	for _, rm := range toRm {
		newLics = Remove(newLics, rm)
	}

	return newLics
}
