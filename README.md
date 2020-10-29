<!--
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
-->

Verifying Licenses
==================

This is an automatic license checker aimed at ensuring compliance with
the Apache Software Foundation processes. It does not ensure compliance,
but it catches many common errors automatically.

To get started quickly, install `weasel` with `go get`:

    go get github.com/comcast/weasel

Then, `cd` into your project's directory and run:

    weasel

That will list all the potentially problematic licenses in the project. You'll
probably need to configure your `.dependency_license` file to document the
dependencies that can't be detected automatically and suppress the ones that
are detected wrongly.

`weasel`
---------

This is the binary with all the logic in it. `weasel` will list every
file in every subdirectory, starting from the target directory, and
determine the most likely license for that file. It errs on the side of
false positives, since the consequences of a false negative are
considerably more serious.

`weasel [-q] [--] <target_dir>`:

  - `-a` Print all files and their licenses, not just problematic files.
  - `-q` Suppress the printing of non-problematic files. This is the default.
  - `-d <sub_dir>` Only run on files in the specified subdirectory.
  - `-f <out_file>` Also write license results to `<out_file>`.
  - `--` Nothing after this is interpreted as an argument.
  - `<target_dir>` To run `weasel` against a different target. The
    target directory must be the root of the project. If it is omitted,
    `weasel` will search directories upward from the current directory,
    looking for a `.git` folder to indicate the root.

`LICENSE`
---------

The `LICENSE` file is at the root of the project (whence you ought to run
`weasel`). This must comply with the requirements of the Apache Software
Foundation and is intended for human consumption.

Nevertheless, with a bit of careful writing, it's possible to have
`weasel` help verify that everything gets covered.

Lines that begin with an `@`-symbol are interpreted as a path
specification that describes a set of files covered by the license.
`weasel` does not validate that `@` files are actually licensed
correctly, merely that they are mentioned. This covers the most common
case, which is adding a (even potentially correctly licensed!) file and
forgetting to mention it in the `LICENSE` file.

Likewise, it's impermissible to use an `@`-line that describes no files.
This usually happens when a dependency is removed and the `LICENSE` file
does not get updated properly.

`@`-lines are interpreted by
[path.Match](https://golang.org/pkg/path/#Match), the syntax for which
is:

    pattern:
        { term }
    term:
        '*'         matches any sequence of non-/ characters
        '?'         matches any single non-/ character
        '[' [ '^' ] { character-range } ']'
                    character class (must be non-empty)
        c           matches character c (c != '*', '?', '\\', '[')
        '\\' c      matches character c

    character-range:
        c           matches character c (c != '\\', '-', ']')
        '\\' c      matches character c
        lo '-' hi   matches character c for lo <= c <= hi

`.dependency_license`
---------------------

Sometimes, there's no reasonable way to automatically detect the
appropriate license for a file, especially files that don't support
comments or are binary. `.dependency_license` allows you to document the
exceptions so that new files show up clearly.

The `.dependency_license` must appear in the root of the project.

Each line should either be empty, a comment (prepended by an octothorp),
or a license exception line. A license exception line is a regular
expression, a comma, then the name of a license, then optionally an
octothorp followed by a comment (which may not contain a comma!).

    license-exception:
        regex ',' license-name [ '#' { commentable-char } ]       Associates the license with the file.        
        regex ',' '!' license-name [ '#' { commentable-char } ]   Disassociates the license from the file.

    regex: A regular expression accepted by golang regexps, described here: https://golang.org/s/re2syntax

    license-name:
        'Apache'    Apache License
        'BSD'       Berkeley Software Distribution License
        'MIT'       Massachusetts Institute of Technology License
        'GoBSD'     BSD-style license used by the GoLang team
        'ISC'       Internet Systems Consortium
        'X11'       MIT License, by an older name.
        'WTFPL'     Do What the Fuck You Want to Public License
        'GPL/LGPL'  Either the GNU General Public License or the GNU Lesser General Public License
        'PD'        Public Domain
        'Docs'      A documentation file
        'Empty'     An empty file
        'Ignored'   A file that ought not be analyzed for compliance

    commentable-char: Any character other than a ','

You can also create a `.dependency_licenses` directory, and all files inside will be used as overrides, with their paths applied to the parent directory.

Output
------

`weasel` prints out problematic files with an exclamation point after the license name. If no licensing or file information could be determined, `Unknown!` is printed as the license. If the general file type is determined `Unknown-type!` is used as the license.

If `weasel` determines the type of the file and the type isn't `Apache-2.0`, then it needs to appear explicitly in the LICENSE file. So a `!` after a recognized license means that the file doesn't match one of the entries in the main LICENSE file.

If the license contains a `~`, it means licensing was determined by looking at a LICENSE file in the same directory. This helps with licensing for vendored dependencies.

Docker Image
------------

A docker image is published for weasel to facilitate use on platforms
without a functional go installation and for users that do not wish to
compile it. You can get up and going immediately with weasel like this:

    docker run --rm -v $(git rev-parse --show-toplevel 2>/dev/null || pwd):/src licenseweasel/weasel /src

Since weasel is running in a container, you have to mount a volume with the
source in it. You can also use weasel via `docker-compose`.

### Docker Image Licensing

Source code for `weasel` is licensed under Apache-2.0; available in the LICENSE
file in this repository.

The image is built from scratch and includes git and it's dependencies glibc
and zlib. Their licenses can be found here:

  - `git`: GPLv2, source repo at https://github.com/git/git
  - `glibc`: LGPLv2.1, releases available at https://www.gnu.org/software/libc/
  - `zlib`: zlib License, releases available at https://www.zlib.net/

You can export the sources tarball with full source for the dependency from the
source image (licenseweasel/weasel-src) with:

```
export TMP_CONTAINER="$(docker create licenseweasel/weasel-src:latest)"
docker export $TMP_CONTAINER | tar -x sources.tgz
docker rm $TMP_CONTAINER
```

Full licenses for these components are included in the source tarball.

GitHub Action
-------------

`weasel` runs as a GitHub action as-is. You can easily and automatically
run it on pushes and pull request by creating a file at
`.github/workflows/weasel.yml` with these contents:

```.yaml
name: Weasel License Validation
on:
  push:
  create:
  pull_request:
    types:
    - opened
    - reopened
    - edited
    - synchronize
jobs:
  weasel:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout
      uses: actions/checkout@v2
    - name: Run weasel
      uses: docker://licenseweasel/weasel:v0.4
```

Of course, be sure to put the appropriate licensing information for your
project as necessary.

Best Practices
--------------

License management can be tricky at the best of times. The goal of this
tool is to automate as much of that as possible. Here are some best
practices:

-   **`weasel` before you commit.** If it prints anything, you
    probably need to add license headers.
-   **Do not `Ignore` files.** If it's reasonable to quiet `weasel`
    about a false positive or negative in another way, do that instead.
-   **If an unrecognized file has a header, update `weasel`, not
    `.dependency_license`.** It's relatively straightforward to add
    license recognition to `licenseList.go`. Doing it that way benefits
    future files as well.
-   **Run `weasel` as part of Continuous Integration.** Issues
    are not usually difficult to fix, but automatic running allows them
    to be fixed promptly. If you use GitHub, see above for easy
    integration with GitHub Actions.


Building Weasel Binaries
--------------
1. Ensure the `VERSION` file reflects the target release version info
2. `docker-compose -f docker-compose.build_bin.yml build --no-cache && docker-compose -f docker-compose.build_bin.yml up`

Your ./dist directory should be filled with archives of binaries and source tarballs
