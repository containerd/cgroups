module github.com/containerd/cgroups/cmd

go 1.18

replace github.com/containerd/cgroups/v3 => ../

require (
	github.com/containerd/cgroups/v3 v3.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.8.1
	github.com/urfave/cli v1.22.5
)

require (
	github.com/cilium/ebpf v0.9.1 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0-20190314233015-f79a8a8ca69d // indirect
	github.com/godbus/dbus/v5 v5.0.4 // indirect
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
