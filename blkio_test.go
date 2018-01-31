package cgroups

import (
	"strings"
	"testing"
)

const data = `   7       0 loop0 0 0 0 0 0 0 0 0 0 0 0
   7       1 loop1 0 0 0 0 0 0 0 0 0 0 0
   7       2 loop2 0 0 0 0 0 0 0 0 0 0 0
   7       3 loop3 0 0 0 0 0 0 0 0 0 0 0
   7       4 loop4 0 0 0 0 0 0 0 0 0 0 0
   7       5 loop5 0 0 0 0 0 0 0 0 0 0 0
   7       6 loop6 0 0 0 0 0 0 0 0 0 0 0
   7       7 loop7 0 0 0 0 0 0 0 0 0 0 0
   8       0 sda 1892042 187697 63489222 1246284 1389086 2887005 134903104 11390608 1 1068060 12692228
   8       1 sda1 1762875 37086 61241570 1200512 1270037 2444415 131214808 11152764 1 882624 12409308
   8       2 sda2 2 0 4 0 0 0 0 0 0 0 0
   8       5 sda5 129102 150611 2244440 45716 18447 442590 3688296 67268 0 62584 112984`

func TestGetDevices(t *testing.T) {
	r := strings.NewReader(data)
	devices, err := getDevices(r)
	if err != nil {
		t.Fatal(err)
	}
	name, ok := devices[deviceKey{8, 0}]
	if !ok {
		t.Fatal("no device found for 8,0")
	}
	const expected = "/dev/sda"
	if name != expected {
		t.Fatalf("expected device name %q but received %q", expected, name)
	}
}
