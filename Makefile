#   Copyright The containerd Authors.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

PACKAGES=$(shell go list ./... | grep -v /vendor/)
GO_TAGS=$(if $(GO_BUILDTAGS),-tags "$(strip $(GO_BUILDTAGS))",)
GO ?= go
GO_BUILD_FLAGS ?=

all: cgutil
	$(GO) build -v $(GO_TAGS)

cgutil:
	cd cmd/cgctl && $(GO) build $(GO_BUILD_FLAGS) -v $(GO_TAGS)

# Follow GNU's standards
# https://www.gnu.org/prep/standards/html_node/Standard-Targets.html
maintainer-clean:
	find cgroup1 cgroup2 \( -name '*.pb.go' -o -name '*.pb.txt' \) -delete

proto:
	buf generate
	@# Keep them Go-idiomatic and backward-compatible with the gogo/protobuf era.
	go-fix-acronym -w -a '(Cpu|Tcp|Rss|Psi)' cgroup1/stats/metrics.pb.go cgroup2/stats/metrics.pb.go
	buf build --exclude-source-info --path cgroup1/stats/metrics.proto -o cgroup1/stats/metrics.pb.txt#format=txtpb
	buf build --exclude-source-info --path cgroup2/stats/metrics.proto -o cgroup2/stats/metrics.pb.txt#format=txtpb

.PHONY: all cgutil maintainer-clean proto
