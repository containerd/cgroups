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

package cgroups

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func TestPids(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	pids := NewPids(mock.root)
	if pids == nil {
		t.Fatal("pids is nil")
	}
	resources := specs.LinuxResources{}
	resources.Pids = &specs.LinuxPids{}
	resources.Pids.Limit = int64(10)
	err = pids.Create("test", &resources)
	if err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(mock.root, "pids", "test", "pids.current")
	if err = ioutil.WriteFile(
		current,
		[]byte(strconv.Itoa(5)),
		defaultFilePerm,
	); err != nil {
		t.Fatal(err)
	}
	metrics := Metrics{}
	err = pids.Stat("test", &metrics)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Pids.Limit != uint64(10) {
		t.Fatalf("expected pids limit %q but received %q",
			uint64(10), metrics.Pids.Limit)
	}
	if metrics.Pids.Current != uint64(5) {
		t.Fatalf("expected pids limit %q but received %q",
			uint64(5), metrics.Pids.Current)
	}
	resources.Pids.Limit = int64(15)
	err = pids.Update("test", &resources)
	if err != nil {
		t.Fatal(err)
	}
	err = pids.Stat("test", &metrics)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Pids.Limit != uint64(15) {
		t.Fatalf("expected pids limit %q but received %q",
			uint64(15), metrics.Pids.Limit)
	}
	if metrics.Pids.Current != uint64(5) {
		t.Fatalf("expected pids limit %q but received %q",
			uint64(5), metrics.Pids.Current)
	}
}

func TestPidsMissingCurrent(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	pids := NewPids(mock.root)
	if pids == nil {
		t.Fatal("pids is nil")
	}
	metrics := Metrics{}
	err = pids.Stat("test", &metrics)
	if err == nil {
		t.Fatal("expected not nil err")
	}
}

func TestPidsMissingMax(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	pids := NewPids(mock.root)
	if pids == nil {
		t.Fatal("pids is nil")
	}
	err = os.Mkdir(filepath.Join(mock.root, "pids", "test"), defaultDirPerm)
	if err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(mock.root, "pids", "test", "pids.current")
	if err = ioutil.WriteFile(
		current,
		[]byte(strconv.Itoa(5)),
		defaultFilePerm,
	); err != nil {
		t.Fatal(err)
	}
	metrics := Metrics{}
	err = pids.Stat("test", &metrics)
	if err == nil {
		t.Fatal("expected not nil err")
	}
}

func TestPidsOverflowMax(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	pids := NewPids(mock.root)
	if pids == nil {
		t.Fatal("pids is nil")
	}
	err = os.Mkdir(filepath.Join(mock.root, "pids", "test"), defaultDirPerm)
	if err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(mock.root, "pids", "test", "pids.current")
	if err = ioutil.WriteFile(
		current,
		[]byte(strconv.Itoa(5)),
		defaultFilePerm,
	); err != nil {
		t.Fatal(err)
	}
	max := filepath.Join(mock.root, "pids", "test", "pids.max")
	bytes, err := hex.DecodeString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	if err != nil {
		t.Fatal(err)
	}
	if err = ioutil.WriteFile(
		max,
		bytes,
		defaultFilePerm,
	); err != nil {
		t.Fatal(err)
	}
	metrics := Metrics{}
	err = pids.Stat("test", &metrics)
	if err == nil {
		t.Fatal("expected not nil err")
	}
}
