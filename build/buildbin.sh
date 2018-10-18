# Copyright 2017 Comcast Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/usr/bin/env bash
set -exuo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
pushd $DIR/..

go build -v
WEASEL_VERSION=$(./weasel -v)

# Take a snapshot of the Source
mkdir -p dist
zip -r dist/weasel-$WEASEL_VERSION-Source.zip ../weasel -x dist/**\*
tar --exclude dist -czvf dist/weasel-$WEASEL_VERSION-Source.tgz *

# Cross compile
GOBINEXT=""
for GOOS in darwin linux windows; do
  for GOARCH in 386 amd64; do
    if [[ "$GOOS" == "windows" ]]
    then
      GOBINEXT=".exe"
    else
      GOBINEXT=""
    fi
    GOOS=$GOOS GOARCH=$GOARCH go build -v
    zip -r dist/weasel-$WEASEL_VERSION-$GOOS-$GOARCH.zip weasel$GOBINEXT
  done
done

# ARM Cross compile because we can
GOOS=linux
GOARCH=arm
for GOARM in 5 6 7; do
  GOOS=$GOOS GOARCH=$GOARCH GOARM=$GOARM go build -v
  zip -r dist/weasel-$WEASEL_VERSION-$GOOS-$GOARCH-ARM$GOARM.zip weasel
done
GOARCH=arm64
GOOS=$GOOS GOARCH=$GOARCH go build -v
zip -r dist/weasel-$WEASEL_VERSION-$GOOS-$GOARCH-ARM8.zip weasel
popd
