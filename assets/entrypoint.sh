#!/bin/bash
#
# entrypoint.sh
#
# Copyright (c) 2017 Junpei Kawamoto
#
# This file is part of Roadie Azure.
#
# Roadie Azure is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Roadie Azure is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Roadie Azure. If not, see <http://www.gnu.org/licenses/>.
#

# This template is an entrypoint of a docker container to execute run steps.
#
if [[ $# != 0 ]]; then
  exec $@
fi

{{range $index, $elements := .Run}}
echo "{{.}}"
sh -c "{{.}}" > /tmp/stdout{{$index}}.txt
{{end}}
