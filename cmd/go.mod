module github.com/containerd/cgroups/cmd

go 1.22.0

replace github.com/containerd/cgroups/v3 => ../

require (
	github.com/containerd/cgroups/v3 v3.0.0-00010101000000-000000000000
	github.com/containerd/log v0.1.0
	github.com/urfave/cli v1.22.16
)

require (
	github.com/cilium/ebpf v0.16.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/opencontainers/runtime-spec v1.3.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/exp v0.0.0-20241108190413-2d47ceb2692f // indirect
	golang.org/x/sys v0.27.0 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
)
