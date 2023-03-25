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

package cgroup1

import (
	"bytes"
	"testing"
)

func BenchmarkReaduint64(b *testing.B) {
	b.ReportAllocs()

	buf := bytes.NewBuffer(nil)
	for i := 0; i < b.N; i++ {
		_, err := readUint("testdata/uint", buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}
