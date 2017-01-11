package cgroups

import "sync"

type Stats struct {
	cpuMu sync.Mutex

	Hugetlb map[string]HugetlbStat
	Pids    *PidsStat
	Cpu     *CpuStat
	Memory  *MemoryStat
	Blkio   *BlkioStat
}

type HugetlbStat struct {
	Usage   uint64
	Max     uint64
	Failcnt uint64
}

type PidsStat struct {
	Current uint64
	Max     uint64
}

type CpuStat struct {
	Usage      CpuUsage
	Throttling Throttle
	Cpus       string
	Mems       string
}

type CpuUsage struct {
	// Units: nanoseconds.
	Total  uint64
	Percpu []uint64
	Kernel uint64
	User   uint64
}

type Throttle struct {
	Periods          uint64
	ThrottledPeriods uint64
	ThrottledTime    uint64
}

type MemoryStat struct {
	Cache     uint64
	Usage     MemoryEntry
	Swap      MemoryEntry
	Kernel    MemoryEntry
	KernelTCP MemoryEntry
	Raw       map[string]uint64
}

type MemoryEntry struct {
	Limit   uint64
	Usage   uint64
	Max     uint64
	Failcnt uint64
}

type BlkioStat struct {
	IoServiceBytesRecursive []BlkioEntry
	IoServicedRecursive     []BlkioEntry
	IoQueuedRecursive       []BlkioEntry
	IoServiceTimeRecursive  []BlkioEntry
	IoWaitTimeRecursive     []BlkioEntry
	IoMergedRecursive       []BlkioEntry
	IoTimeRecursive         []BlkioEntry
	SectorsRecursive        []BlkioEntry
}

type BlkioEntry struct {
	Op    string
	Major uint64
	Minor uint64
	Value uint64
}
