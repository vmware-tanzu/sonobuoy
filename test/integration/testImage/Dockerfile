# Copyright 2019 Sonobuoy project contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.18 AS base
WORKDIR /src

# Handle the go modules first to take advantage of Docker cache.
COPY src/go.mod .
COPY src/go.sum .
RUN go mod download

# Get the rest of the files and build.
COPY src .
RUN CGO_ENABLED=0 go build -o /go/bin/testImage ./...

FROM gcr.io/distroless/static-debian10:latest
WORKDIR /
COPY --from=base /go/bin/testImage /testImage
COPY resources /resources
CMD [ "/testImage" ]

