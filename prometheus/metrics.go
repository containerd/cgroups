package prometheus

import (
	"strconv"

	"github.com/crosbymichael/cgroups"
	metrics "github.com/docker/go-metrics"
)

func New(ns *metrics.Namespace) *Collector {
	// add machine cpus and memory info
	return &Collector{
		pidsLimit:   ns.NewLabeledGauge("pids", "The maximum number of pids allowed", metrics.Unit("limit"), "id"),
		pidsCurrent: ns.NewLabeledGauge("pids", "The current number of pids", metrics.Unit("current"), "id"),

		cpuTotal:            ns.NewLabeledGauge("cpu_total", "The total cpu time", metrics.Nanoseconds, "id"),
		cpuKernel:           ns.NewLabeledGauge("cpu_kernel", "The total kernel cpu time", metrics.Nanoseconds, "id"),
		cpuUser:             ns.NewLabeledGauge("cpu_user", "The total user cpu time", metrics.Nanoseconds, "id"),
		cpuPerCpu:           ns.NewLabeledGauge("per_cpu", "The total cpu time per cpu", metrics.Nanoseconds, "id", "core"),
		cpuThrottlePeriods:  ns.NewLabeledGauge("cpu_throttle_periods", "The total throttle periods", metrics.Total, "id"),
		cpuThrottledPeriods: ns.NewLabeledGauge("cpu_throttled_periods", "The total throttled periods", metrics.Total, "id"),
		cpuThrottledTime:    ns.NewLabeledGauge("cpu_throttled_time", "The total throttled time", metrics.Nanoseconds, "id"),

		memoryCache: ns.NewLabeledGauge("memory_cache", "The cache amount used by the container", metrics.Bytes, "id"),

		memoryUsageFail:  ns.NewLabeledGauge("memory_usage_fail", "The usage failcnt", metrics.Total, "id"),
		memoryUsageLimit: ns.NewLabeledGauge("memory_usage_limit", "The memory limit", metrics.Bytes, "id"),
		memoryUsageMax:   ns.NewLabeledGauge("memory_usage_max", "The max memory used", metrics.Bytes, "id"),
		memoryUsageUsage: ns.NewLabeledGauge("memory_usage_usage", "The current memory used", metrics.Bytes, "id"),

		memorySwapFail:  ns.NewLabeledGauge("memory_swap_fail", "The swap failcnt", metrics.Total, "id"),
		memorySwapLimit: ns.NewLabeledGauge("memory_swap_limit", "The swap limit", metrics.Bytes, "id"),
		memorySwapMax:   ns.NewLabeledGauge("memory_swap_max", "The swap memory used", metrics.Bytes, "id"),
		memorySwapUsage: ns.NewLabeledGauge("memory_swap_usage", "The swap memory used", metrics.Bytes, "id"),

		memoryKernelFail:  ns.NewLabeledGauge("memory_kernel_fail", "The kernel failcnt", metrics.Total, "id"),
		memoryKernelLimit: ns.NewLabeledGauge("memory_kernel_limit", "The kernel limit", metrics.Bytes, "id"),
		memoryKernelMax:   ns.NewLabeledGauge("memory_kernel_max", "The kernel memory used", metrics.Bytes, "id"),
		memoryKernelUsage: ns.NewLabeledGauge("memory_kernel_usage", "The kernel memory used", metrics.Bytes, "id"),

		memoryKernelTCPFail:  ns.NewLabeledGauge("memory_kernel_tcp_fail", "The kernel tcp failcnt", metrics.Total, "id"),
		memoryKernelTCPLimit: ns.NewLabeledGauge("memory_kernel_tcp_limit", "The kernel tcp limit", metrics.Bytes, "id"),
		memoryKernelTCPMax:   ns.NewLabeledGauge("memory_kernel_tcp_max", "The kernel tcp memory used", metrics.Bytes, "id"),
		memoryKernelTCPUsage: ns.NewLabeledGauge("memory_kernel_tcp_usage", "The kernel tcp memory used", metrics.Bytes, "id"),

		hugetlbUsage: ns.NewLabeledGauge("hugetlb_usage", "The hugetlb usage", metrics.Bytes, "id", "page"),
		hugetlbFail:  ns.NewLabeledGauge("hugetlb_fail", "The hugetlb failcnt", metrics.Total, "id", "page"),
		hugetlbMax:   ns.NewLabeledGauge("hugetlb_max", "The hugetlb max usage", metrics.Bytes, "id", "page"),

		blkioIOMergedRecursive:       ns.NewLabeledGauge("blkio_io_merged_recursive", "The blkio io merged recursive", metrics.Total, "id", "op", "major", "minor"),
		blkioIOQueuedRecursive:       ns.NewLabeledGauge("blkio_io_queued_recursive", "The blkio io queued recursive", metrics.Total, "id", "op", "major", "minor"),
		blkioIOServiceBytesRecursive: ns.NewLabeledGauge("blkio_io_service_bytes_recursive", "The blkio io service bytes recursive", metrics.Bytes, "id", "op", "major", "minor"),
		blkioIOServiceTimeRecursive:  ns.NewLabeledGauge("blkio_io_service_time_recursive", "The blkio io service time recursive", metrics.Total, "id", "op", "major", "minor"),
		blkioIOServiedRecursive:      ns.NewLabeledGauge("blkio_io_serviced_recursive", "The blkio io serviced recursive", metrics.Total, "id", "op", "major", "minor"),
		blkioIOTimeRecursive:         ns.NewLabeledGauge("blkio_io_time_recursive", "The blkio io time recursive", metrics.Total, "id", "op", "major", "minor"),
		blkioSectorsRecursive:        ns.NewLabeledGauge("blkio_sectors_recursive", "The blkio sectors recursive", metrics.Total, "id", "op", "major", "minor"),
	}
}

// Collector provides the ability to collect container stats and export
// them in the prometheus format
type Collector struct {
	pidsLimit   metrics.LabeledGauge
	pidsCurrent metrics.LabeledGauge

	cpuTotal            metrics.LabeledGauge
	cpuKernel           metrics.LabeledGauge
	cpuUser             metrics.LabeledGauge
	cpuPerCpu           metrics.LabeledGauge
	cpuThrottlePeriods  metrics.LabeledGauge
	cpuThrottledPeriods metrics.LabeledGauge
	cpuThrottledTime    metrics.LabeledGauge

	memoryCache metrics.LabeledGauge

	memoryUsageLimit metrics.LabeledGauge
	memoryUsageUsage metrics.LabeledGauge
	memoryUsageMax   metrics.LabeledGauge
	memoryUsageFail  metrics.LabeledGauge

	memorySwapLimit metrics.LabeledGauge
	memorySwapUsage metrics.LabeledGauge
	memorySwapMax   metrics.LabeledGauge
	memorySwapFail  metrics.LabeledGauge

	memoryKernelLimit metrics.LabeledGauge
	memoryKernelUsage metrics.LabeledGauge
	memoryKernelMax   metrics.LabeledGauge
	memoryKernelFail  metrics.LabeledGauge

	memoryKernelTCPLimit metrics.LabeledGauge
	memoryKernelTCPUsage metrics.LabeledGauge
	memoryKernelTCPMax   metrics.LabeledGauge
	memoryKernelTCPFail  metrics.LabeledGauge

	hugetlbUsage metrics.LabeledGauge
	hugetlbFail  metrics.LabeledGauge
	hugetlbMax   metrics.LabeledGauge

	blkioIOServiceBytesRecursive metrics.LabeledGauge
	blkioIOServiedRecursive      metrics.LabeledGauge
	blkioIOQueuedRecursive       metrics.LabeledGauge
	blkioIOServiceTimeRecursive  metrics.LabeledGauge
	blkioIOMergedRecursive       metrics.LabeledGauge
	blkioIOTimeRecursive         metrics.LabeledGauge
	blkioSectorsRecursive        metrics.LabeledGauge
}

func (c *Collector) Collect(id string, cg cgroups.Cgroup) error {
	stats, err := cg.Stat()
	if err != nil {
		return err
	}
	if stats.Pids != nil {
		c.pidsLimit.WithValues(id).Set(float64(stats.Pids.Limit))
		c.pidsCurrent.WithValues(id).Set(float64(stats.Pids.Current))
	}
	if stats.Cpu != nil {
		c.cpuTotal.WithValues(id).Set(float64(stats.Cpu.Usage.Total))
		c.cpuKernel.WithValues(id).Set(float64(stats.Cpu.Usage.Kernel))
		c.cpuUser.WithValues(id).Set(float64(stats.Cpu.Usage.User))
		// set per cpu usage
		for i, cpu := range stats.Cpu.Usage.PerCpu {
			c.cpuPerCpu.WithValues(id, strconv.Itoa(i)).Set(float64(cpu))
		}
		c.cpuThrottlePeriods.WithValues(id).Set(float64(stats.Cpu.Throttling.Periods))
		c.cpuThrottledPeriods.WithValues(id).Set(float64(stats.Cpu.Throttling.ThrottledPeriods))
		c.cpuThrottledTime.WithValues(id).Set(float64(stats.Cpu.Throttling.ThrottledTime))
	}
	if stats.Memory != nil {
		c.memoryCache.WithValues(id).Set(float64(stats.Memory.Cache))

		c.memoryUsageFail.WithValues(id).Set(float64(stats.Memory.Usage.Failcnt))
		c.memoryUsageLimit.WithValues(id).Set(float64(stats.Memory.Usage.Limit))
		c.memoryUsageMax.WithValues(id).Set(float64(stats.Memory.Usage.Max))
		c.memoryUsageUsage.WithValues(id).Set(float64(stats.Memory.Usage.Usage))

		c.memorySwapFail.WithValues(id).Set(float64(stats.Memory.Swap.Failcnt))
		c.memorySwapLimit.WithValues(id).Set(float64(stats.Memory.Swap.Limit))
		c.memorySwapMax.WithValues(id).Set(float64(stats.Memory.Swap.Max))
		c.memorySwapUsage.WithValues(id).Set(float64(stats.Memory.Swap.Usage))

		c.memoryKernelFail.WithValues(id).Set(float64(stats.Memory.Kernel.Failcnt))
		c.memoryKernelLimit.WithValues(id).Set(float64(stats.Memory.Kernel.Limit))
		c.memoryKernelMax.WithValues(id).Set(float64(stats.Memory.Kernel.Max))
		c.memoryKernelUsage.WithValues(id).Set(float64(stats.Memory.Kernel.Usage))

		c.memoryKernelTCPFail.WithValues(id).Set(float64(stats.Memory.KernelTCP.Failcnt))
		c.memoryKernelTCPLimit.WithValues(id).Set(float64(stats.Memory.KernelTCP.Limit))
		c.memoryKernelTCPMax.WithValues(id).Set(float64(stats.Memory.KernelTCP.Max))
		c.memoryKernelTCPUsage.WithValues(id).Set(float64(stats.Memory.KernelTCP.Usage))
	}
	if stats.Hugetlb != nil {
		for k, v := range stats.Hugetlb {
			c.hugetlbUsage.WithValues(id, k).Set(float64(v.Usage))
			c.hugetlbFail.WithValues(id, k).Set(float64(v.Failcnt))
			c.hugetlbMax.WithValues(id, k).Set(float64(v.Max))
		}
	}
	if stats.Blkio != nil {
		for _, v := range []struct {
			gauge   metrics.LabeledGauge
			entries []cgroups.BlkioEntry
		}{
			{
				gauge:   c.blkioIOMergedRecursive,
				entries: stats.Blkio.IoMergedRecursive,
			},
			{
				gauge:   c.blkioIOQueuedRecursive,
				entries: stats.Blkio.IoQueuedRecursive,
			},
			{
				gauge:   c.blkioIOServiceBytesRecursive,
				entries: stats.Blkio.IoServiceBytesRecursive,
			},
			{
				gauge:   c.blkioIOServiceTimeRecursive,
				entries: stats.Blkio.IoServiceTimeRecursive,
			},
			{
				gauge:   c.blkioIOServiedRecursive,
				entries: stats.Blkio.IoServicedRecursive,
			},
			{
				gauge:   c.blkioIOTimeRecursive,
				entries: stats.Blkio.IoTimeRecursive,
			},
			{
				gauge:   c.blkioSectorsRecursive,
				entries: stats.Blkio.SectorsRecursive,
			},
		} {
			setIO(id, v.gauge, v.entries)
		}
	}
	return nil
}

func setIO(id string, gauge metrics.LabeledGauge, entries []cgroups.BlkioEntry) {
	for _, i := range entries {
		gauge.WithValues(id, i.Op, strconv.FormatUint(i.Major, 10), strconv.FormatUint(i.Minor, 10)).Set(float64(i.Value))
	}
}
