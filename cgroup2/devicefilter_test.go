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
	"strings"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func hash(s, comm string) string {
	var res []string
	for _, l := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, comm) {
			continue
		}
		res = append(res, trimmed)
	}
	return strings.Join(res, "\n")
}

func testDeviceFilter(t testing.TB, devices []specs.LinuxDeviceCgroup, expectedStr string) {
	insts, _, err := DeviceFilter(devices)
	require.NoErrorf(t, err, "%s: (devices: %+v)", t.Name(), devices)

	s := insts.String()
	t.Logf("%s: devices: %+v\n%s", t.Name(), devices, s)
	if expectedStr != "" {
		hashed := hash(s, "//")
		expectedHashed := hash(expectedStr, "//")
		require.Equal(t, expectedHashed, hashed)
	}
}

func TestDeviceFilter_Nil(t *testing.T) {
	expected := `
// load parameters into registers
        0: LdXMemH dst: r2 src: r1 off: 0 imm: 0
        1: LdXMemW dst: r3 src: r1 off: 0 imm: 0
        2: RSh32Imm dst: r3 imm: 16
        3: LdXMemW dst: r4 src: r1 off: 4 imm: 0
        4: LdXMemW dst: r5 src: r1 off: 8 imm: 0
block-0:
// return 0 (reject)
        5: Mov32Imm dst: r0 imm: 0
        6: Exit
	`
	testDeviceFilter(t, nil, expected)
}

func TestDeviceFilter_Privileged(t *testing.T) {
	devices := []specs.LinuxDeviceCgroup{
		{
			Type:   "a",
			Major:  pointerInt64(-1),
			Minor:  pointerInt64(-1),
			Access: "rwm",
			Allow:  true,
		},
	}
	expected := `
// load parameters into registers
        0: LdXMemH dst: r2 src: r1 off: 0 imm: 0
        1: LdXMemW dst: r3 src: r1 off: 0 imm: 0
        2: RSh32Imm dst: r3 imm: 16
        3: LdXMemW dst: r4 src: r1 off: 4 imm: 0
        4: LdXMemW dst: r5 src: r1 off: 8 imm: 0
block-0:
// return 1 (accept)
        5: Mov32Imm dst: r0 imm: 1
        6: Exit
	`
	testDeviceFilter(t, devices, expected)
}

func TestDeviceFilter_PrivilegedExceptSingleDevice(t *testing.T) {
	devices := []specs.LinuxDeviceCgroup{
		{
			Type:   "a",
			Major:  pointerInt64(-1),
			Minor:  pointerInt64(-1),
			Access: "rwm",
			Allow:  true,
		},
		{
			Type:   "b",
			Major:  pointerInt64(8),
			Minor:  pointerInt64(0),
			Access: "rwm",
			Allow:  false,
		},
	}
	expected := `
// load parameters into registers
         0: LdXMemH dst: r2 src: r1 off: 0 imm: 0
         1: LdXMemW dst: r3 src: r1 off: 0 imm: 0
         2: RSh32Imm dst: r3 imm: 16
         3: LdXMemW dst: r4 src: r1 off: 4 imm: 0
         4: LdXMemW dst: r5 src: r1 off: 8 imm: 0
block-0:
// return 0 (reject) if type==b && major == 8 && minor == 0
         5: JNEImm dst: r2 off: -1 imm: 1 <block-1>
         6: JNEImm dst: r4 off: -1 imm: 8 <block-1>
         7: JNEImm dst: r5 off: -1 imm: 0 <block-1>
         8: Mov32Imm dst: r0 imm: 0
         9: Exit
block-1:
// return 1 (accept)
        10: Mov32Imm dst: r0 imm: 1
        11: Exit
`
	testDeviceFilter(t, devices, expected)
}

func TestDeviceFilter_Weird(t *testing.T) {
	devices := []specs.LinuxDeviceCgroup{
		{
			Type:   "b",
			Major:  pointerInt64(8),
			Minor:  pointerInt64(1),
			Access: "rwm",
			Allow:  false,
		},
		{
			Type:   "a",
			Major:  pointerInt64(-1),
			Minor:  pointerInt64(-1),
			Access: "rwm",
			Allow:  true,
		},
		{
			Type:   "b",
			Major:  pointerInt64(8),
			Minor:  pointerInt64(2),
			Access: "rwm",
			Allow:  false,
		},
	}
	// 8/1 is allowed, 8/2 is not allowed.
	// This conforms to runc v1.0.0-rc.9 (cgroup1) behavior.
	expected := `
// load parameters into registers
         0: LdXMemH dst: r2 src: r1 off: 0 imm: 0
         1: LdXMemW dst: r3 src: r1 off: 0 imm: 0
         2: RSh32Imm dst: r3 imm: 16
         3: LdXMemW dst: r4 src: r1 off: 4 imm: 0
         4: LdXMemW dst: r5 src: r1 off: 8 imm: 0
block-0:
// return 0 (reject) if type==b && major == 8 && minor == 2
         5: JNEImm dst: r2 off: -1 imm: 1 <block-1>
         6: JNEImm dst: r4 off: -1 imm: 8 <block-1>
         7: JNEImm dst: r5 off: -1 imm: 2 <block-1>
         8: Mov32Imm dst: r0 imm: 0
         9: Exit
block-1:
// return 1 (accept)
        10: Mov32Imm dst: r0 imm: 1
        11: Exit
`
	testDeviceFilter(t, devices, expected)
}

func pointerInt64(int int64) *int64 {
	return &int
}
