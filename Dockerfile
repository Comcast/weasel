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

FROM golang:1.10-alpine AS build-weasel

# Build weasel
WORKDIR /go/src/weasel
COPY . .
RUN CGO_ENABLED=0 go install weasel
RUN CGO_ENABLED=0 go install weasel/dedup

# Build git
FROM debian AS build-git
RUN sed -i 's/^deb\(.*\)$/deb\1\ndeb-src\1/' /etc/apt/sources.list
RUN apt-get -y update && apt-get -y install git build-essential libssl-dev zlib1g-dev libcurl4-openssl-dev libexpat-dev gettext
RUN mkdir -p sources; cd sources; git clone https://github.com/git/git \
	&& cd git \
	&& git checkout v2.26.2 \
	&& make prefix=/opt/git NO_TCLTK=yes all install \
	&& rm -rf .git
RUN mkdir -p sources; cd sources; \
    ldd /opt/git/bin/git | tr -s '[:blank:]' '\n' | grep '^/' | \
    xargs -I % sh -c 'dpkg -S %' | cut -f1 -d: | sort | uniq | \
    xargs -I % apt-get source %=$(apt-cache policy % | grep '^ \*\*\*' | cut '-d ' -f3)
RUN tar -czf /sources.tgz sources
RUN ldd /opt/git/bin/git | tr -s '[:blank:]' '\n' | grep '^/' | \
    xargs -I % sh -c 'mkdir -p $(dirname deps%); cp % deps%;'

# Create empty image with only the required binaries
FROM scratch AS bin
COPY --from=build-weasel /go/bin/weasel /usr/bin/weasel
COPY --from=build-weasel /go/bin/dedup /usr/bin/dedup
COPY --from=build-git /opt/git /opt/git
COPY --from=build-git /deps /
RUN ["/usr/bin/dedup", "/opt/git/libexec/git-core"]
ENV PATH="/opt/git/bin:${PATH}"
ENTRYPOINT ["/usr/bin/weasel"]
CMD []

# Create an image just like bin, but add sources for all dependencies
FROM bin AS src
COPY --from=build-git /sources.tgz /sources.tgz
