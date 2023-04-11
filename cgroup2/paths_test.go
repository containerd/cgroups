/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package cgroup2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyGroupPath(t *testing.T) {
	valids := map[string]bool{
		"/":                          true,
		"":                           false,
		"/foo":                       true,
		"/foo/bar":                   true,
		"/sys/fs/cgroup/foo":         false,
		"/sys/fs/cgroup/unified/foo": false,
		"foo":                        false,
		"/foo/../bar":                false,
	}
	for s, valid := range valids {
		err := VerifyGroupPath(s)
		if valid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}
