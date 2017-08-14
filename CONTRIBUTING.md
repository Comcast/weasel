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
Contributing
============

Contributions are managed most easily via pull requests. This project is
hosted at https://github.com/comcast/weasel . Head over, fork the
project and open a pull request.

Goals
-----

Weasel is not designed as a general-purpose all-encompasing license
checker. Those exist, but are more difficult to use and configure.
Weasel is designed specifically for projects trying to meet Apache
Software Foundation release guidelines.

Likewise, weasel considers false-positives to be better than
false-negatives. It's better to have a file incorrectly identified as
having a restrictive license (which you can then manually override) than
to have it incorrectly identified as having no license or a
non-restrictive license. Since many files may include no more license
information than a single `Licensed GPL`, weasel uses a very aggressive
approach.

Weasel expects that you'll prepare a directory for it and maintain it.
It doesn't expect to run nicely on just any project. There are too many
details to get right and attempting to make it work nicely without
managing the un-automatable files will cause it to quietly miss
potentially infringing licenses.

Weasel is designed to be run automatically, by Continuous Integration,
as well as manually, by either commit-hook or fastidious human.

Dependencies
------------

Weasel is intentionally devoid of dependencies. This makes it easy to
pull and build on any possible system. It's possible that dependencies
may provide significant value in the future, but their use should be
weighed against the potential difficulty they may cause building the
tool.
