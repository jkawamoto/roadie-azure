#
# Makefile
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
# along with Roadie Azure.  If not, see <http:#www.gnu.org/licenses/>.
#
VERSION = snapshot
default: build
.PHONY: asset build release get-deps test

asset:
	rm assets/assets.go
	go-bindata -pkg assets -o assets/assets.go -nometadata assets/*

build:
	goxc -d=pkg -pv=$(VERSION) -os="linux" -arch="amd64"

release:
	ghr -u jkawamoto  v$(VERSION) pkg/$(VERSION)

get-deps:
	go get -d -t -v .

test:
	go test -v ./...
