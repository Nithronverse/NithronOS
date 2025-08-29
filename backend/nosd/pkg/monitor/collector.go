package monitor

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Collector gathers system metrics
type Collector struct {
	logger     zerolog.Logger
	storage    *TimeSeriesStorage
	interval   time.Duration
	mu         sync.RWMutex
	lastValues map[string]interface{}
	cancel     context.CancelFunc
}

// NewCollector creates a new metrics collector
func NewCollector(logger zerolog.Logger, storage *TimeSeriesStorage, interval time.Duration) *Collector {
	if interval == 0 {
		interval = 60 * time.Second
	}

	return &Collector{
		logger:     logger.With().Str("component", "metrics-collector").Logger(),
		storage:    storage,
		interval:   interval,
		lastValues: make(map[string]interface{}),
	}
}

// Start begins metric collection
func (c *Collector) Start(ctx context.Context) error {
	c.logger.Info().Dur("interval", c.interval).Msg("Starting metrics collector")

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// Start collection goroutines
	go c.collectLoop(ctx)

	return nil
}

// Stop halts metric collection
func (c *Collector) Stop() error {
	c.logger.Info().Msg("Stopping metrics collector")

	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

// collectLoop runs the main collection loop
func (c *Collector) collectLoop(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Collect immediately
	c.collect()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

// collect gathers all metrics
func (c *Collector) collect() {
	now := time.Now()

	// Collect different metric types in parallel
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.collectCPU(now)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.collectMemory(now)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.collectLoad(now)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.collectDisk(now)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.collectNetwork(now)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.collectServices(now)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.collectBtrfs(now)
	}()

	wg.Wait()

	c.logger.Debug().Msg("Metrics collection completed")
}

// collectCPU gathers CPU metrics
func (c *Collector) collectCPU(now time.Time) {
	// Overall CPU usage
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get CPU usage")
		return
	}

	if len(percent) > 0 {
		_ = c.storage.Store(MetricTypeCPU, now, percent[0], nil)
		c.updateLastValue("cpu_percent", percent[0])
	}

	// Per-core CPU usage
	perCore, err := cpu.Percent(time.Second, true)
	if err == nil {
		for i, p := range perCore {
			labels := map[string]string{"core": strconv.Itoa(i)}
			_ = c.storage.Store(MetricTypeCPU, now, p, labels)
		}
	}

	// CPU temperature (if available)
	c.collectCPUTemperature(now)
}

// collectCPUTemperature gathers CPU temperature
func (c *Collector) collectCPUTemperature(now time.Time) {
	// Try to read from thermal zones (Linux)
	cmd := exec.Command("sh", "-c", "cat /sys/class/thermal/thermal_zone*/temp 2>/dev/null | head -1")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		if temp, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
			tempC := float64(temp) / 1000.0
			labels := map[string]string{"sensor": "cpu"}
			_ = c.storage.Store(MetricTypeDiskTemp, now, tempC, labels)
		}
	}
}

// collectMemory gathers memory metrics
func (c *Collector) collectMemory(now time.Time) {
	v, err := mem.VirtualMemory()
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get memory info")
		return
	}

	_ = c.storage.Store(MetricTypeMemory, now, v.UsedPercent, nil)
	_ = c.storage.Store(MetricTypeMemory, now, float64(v.Used), map[string]string{"type": "used"})
	_ = c.storage.Store(MetricTypeMemory, now, float64(v.Available), map[string]string{"type": "available"})

	c.updateLastValue("memory_percent", v.UsedPercent)
	c.updateLastValue("memory_used", v.Used)
	c.updateLastValue("memory_total", v.Total)

	// Swap
	s, err := mem.SwapMemory()
	if err == nil && s.Total > 0 {
		_ = c.storage.Store(MetricTypeSwap, now, s.UsedPercent, nil)
		_ = c.storage.Store(MetricTypeSwap, now, float64(s.Used), map[string]string{"type": "used"})

		c.updateLastValue("swap_percent", s.UsedPercent)
		c.updateLastValue("swap_used", s.Used)
		c.updateLastValue("swap_total", s.Total)
	}
}

// collectLoad gathers load average
func (c *Collector) collectLoad(now time.Time) {
	l, err := load.Avg()
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get load average")
		return
	}

	_ = c.storage.Store(MetricTypeLoad, now, l.Load1, map[string]string{"period": "1m"})
	_ = c.storage.Store(MetricTypeLoad, now, l.Load5, map[string]string{"period": "5m"})
	_ = c.storage.Store(MetricTypeLoad, now, l.Load15, map[string]string{"period": "15m"})

	c.updateLastValue("load1", l.Load1)
	c.updateLastValue("load5", l.Load5)
	c.updateLastValue("load15", l.Load15)
}

// collectDisk gathers disk metrics
func (c *Collector) collectDisk(now time.Time) {
	// Disk usage
	partitions, err := disk.Partitions(false)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get disk partitions")
		return
	}

	for _, p := range partitions {
		// Skip pseudo filesystems
		if strings.HasPrefix(p.Fstype, "tmpfs") || p.Fstype == "devtmpfs" {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		labels := map[string]string{
			"device":     p.Device,
			"mountpoint": p.Mountpoint,
			"fstype":     p.Fstype,
		}

		_ = c.storage.Store(MetricTypeDiskSpace, now, usage.UsedPercent, labels)
		_ = c.storage.Store(MetricTypeDiskSpace, now, float64(usage.Used),
			mergeMaps(labels, map[string]string{"type": "used"}))
		_ = c.storage.Store(MetricTypeDiskSpace, now, float64(usage.Free),
			mergeMaps(labels, map[string]string{"type": "free"}))

		// Update last values for dashboard
		if p.Mountpoint == "/" {
			c.updateLastValue("disk_root_percent", usage.UsedPercent)
		}
	}

	// Disk I/O
	ioCounters, err := disk.IOCounters()
	if err == nil {
		for device, io := range ioCounters {
			labels := map[string]string{"device": device}

			_ = c.storage.Store(MetricTypeDiskIO, now, float64(io.ReadBytes),
				mergeMaps(labels, map[string]string{"type": "read_bytes"}))
			_ = c.storage.Store(MetricTypeDiskIO, now, float64(io.WriteBytes),
				mergeMaps(labels, map[string]string{"type": "write_bytes"}))
			_ = c.storage.Store(MetricTypeDiskIO, now, float64(io.ReadCount),
				mergeMaps(labels, map[string]string{"type": "read_ops"}))
			_ = c.storage.Store(MetricTypeDiskIO, now, float64(io.WriteCount),
				mergeMaps(labels, map[string]string{"type": "write_ops"}))
		}
	}

	// Disk temperatures and SMART
	c.collectDiskSMART(now)
}

// collectDiskSMART gathers SMART data
func (c *Collector) collectDiskSMART(now time.Time) {
	// Get list of block devices
	devices, err := exec.Command("lsblk", "-ndo", "NAME,TYPE").Output()
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(devices), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "disk" {
			continue
		}

		device := "/dev/" + fields[0]

		// Run smartctl to get health and temperature
		cmd := exec.Command("smartctl", "-H", "-A", "-j", device)
		output, _ := cmd.Output()

		var smartData struct {
			SmartStatus struct {
				Passed bool `json:"passed"`
			} `json:"smart_status"`
			Temperature struct {
				Current int `json:"current"`
			} `json:"temperature"`
			AtaSmartAttributes struct {
				Table []struct {
					ID    int    `json:"id"`
					Name  string `json:"name"`
					Value int    `json:"value"`
					Raw   struct {
						Value int `json:"value"`
					} `json:"raw"`
				} `json:"table"`
			} `json:"ata_smart_attributes"`
		}

		if err := json.Unmarshal(output, &smartData); err == nil {
			// Temperature
			if smartData.Temperature.Current > 0 {
				labels := map[string]string{"device": device}
				_ = c.storage.Store(MetricTypeDiskTemp, now, float64(smartData.Temperature.Current), labels)
			}

			// SMART health
			health := 0.0
			if smartData.SmartStatus.Passed {
				health = 1.0
			}
			labels := map[string]string{"device": device}
			_ = c.storage.Store(MetricTypeDiskSMART, now, health, labels)

			// Specific attributes
			for _, attr := range smartData.AtaSmartAttributes.Table {
				switch attr.ID {
				case 5: // Reallocated sectors
					labels := map[string]string{"device": device, "attr": "reallocated"}
					_ = c.storage.Store(MetricTypeDiskSMART, now, float64(attr.Raw.Value), labels)
				case 197: // Current pending sectors
					labels := map[string]string{"device": device, "attr": "pending"}
					_ = c.storage.Store(MetricTypeDiskSMART, now, float64(attr.Raw.Value), labels)
				case 198: // Uncorrectable sectors
					labels := map[string]string{"device": device, "attr": "uncorrectable"}
					_ = c.storage.Store(MetricTypeDiskSMART, now, float64(attr.Raw.Value), labels)
				}
			}
		}
	}
}

// collectNetwork gathers network metrics
func (c *Collector) collectNetwork(now time.Time) {
	interfaces, err := net.IOCounters(true)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get network stats")
		return
	}

	for _, iface := range interfaces {
		// Skip loopback
		if iface.Name == "lo" {
			continue
		}

		labels := map[string]string{"interface": iface.Name}

		_ = c.storage.Store(MetricTypeNetworkRX, now, float64(iface.BytesRecv), labels)
		_ = c.storage.Store(MetricTypeNetworkTX, now, float64(iface.BytesSent), labels)
		_ = c.storage.Store(MetricTypeNetworkRX, now, float64(iface.PacketsRecv),
			mergeMaps(labels, map[string]string{"type": "packets"}))
		_ = c.storage.Store(MetricTypeNetworkTX, now, float64(iface.PacketsSent),
			mergeMaps(labels, map[string]string{"type": "packets"}))
		_ = c.storage.Store(MetricTypeNetworkRX, now, float64(iface.Errin),
			mergeMaps(labels, map[string]string{"type": "errors"}))
		_ = c.storage.Store(MetricTypeNetworkTX, now, float64(iface.Errout),
			mergeMaps(labels, map[string]string{"type": "errors"}))
	}
}

// collectServices gathers service health
func (c *Collector) collectServices(now time.Time) {
	services := []string{"nosd", "nos-agent", "caddy", "wireguard", "docker"}

	for _, service := range services {
		cmd := exec.Command("systemctl", "show", service,
			"--property=ActiveState,SubState,MainPID,NRestarts")
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		props := make(map[string]string)
		for _, line := range strings.Split(string(output), "\n") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				props[parts[0]] = parts[1]
			}
		}

		health := 0.0
		if props["ActiveState"] == "active" {
			health = 1.0
		}

		labels := map[string]string{"service": service}
		_ = c.storage.Store(MetricTypeServiceHealth, now, health, labels)

		// Restart count
		if restarts, err := strconv.Atoi(props["NRestarts"]); err == nil {
			labels := map[string]string{"service": service, "type": "restarts"}
			_ = c.storage.Store(MetricTypeServiceHealth, now, float64(restarts), labels)
		}

		// CPU and memory if running
		if pid, err := strconv.Atoi(props["MainPID"]); err == nil && pid > 0 {
			if proc, err := process.NewProcess(int32(pid)); err == nil {
				if cpu, err := proc.CPUPercent(); err == nil {
					labels := map[string]string{"service": service, "type": "cpu"}
					_ = c.storage.Store(MetricTypeServiceHealth, now, cpu, labels)
				}

				if mem, err := proc.MemoryInfo(); err == nil {
					labels := map[string]string{"service": service, "type": "memory"}
					_ = c.storage.Store(MetricTypeServiceHealth, now, float64(mem.RSS), labels)
				}
			}
		}
	}
}

// collectBtrfs gathers Btrfs-specific metrics
func (c *Collector) collectBtrfs(now time.Time) {
	// Get Btrfs filesystems
	mounts, _ := disk.Partitions(false)
	for _, mount := range mounts {
		if mount.Fstype != "btrfs" {
			continue
		}

		// Device stats
		cmd := exec.Command("btrfs", "device", "stats", mount.Mountpoint)
		output, err := cmd.Output()
		if err == nil {
			errors := c.parseBtrfsErrors(string(output))
			labels := map[string]string{"mountpoint": mount.Mountpoint}

			for errorType, count := range errors {
				l := mergeMaps(labels, map[string]string{"error_type": errorType})
				_ = c.storage.Store(MetricTypeBtrfsErrors, now, float64(count), l)
			}
		}

		// Scrub status
		cmd = exec.Command("btrfs", "scrub", "status", mount.Mountpoint)
		output, err = cmd.Output()
		if err == nil {
			scrubInfo := c.parseScrubStatus(string(output))
			labels := map[string]string{"mountpoint": mount.Mountpoint}

			if scrubInfo["running"] == "1" {
				_ = c.storage.Store(MetricTypeBtrfsScrub, now, 1.0,
					mergeMaps(labels, map[string]string{"state": "running"}))
			} else {
				_ = c.storage.Store(MetricTypeBtrfsScrub, now, 0.0,
					mergeMaps(labels, map[string]string{"state": "idle"}))
			}
		}
	}
}

// parseBtrfsErrors parses btrfs device stats output
func (c *Collector) parseBtrfsErrors(output string) map[string]int {
	errors := make(map[string]int)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "write_io_errs") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if val, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					errors["write"] = val
				}
			}
		} else if strings.Contains(line, "read_io_errs") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if val, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					errors["read"] = val
				}
			}
		} else if strings.Contains(line, "corruption_errs") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if val, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					errors["corruption"] = val
				}
			}
		}
	}
	return errors
}

// parseScrubStatus parses btrfs scrub status output
func (c *Collector) parseScrubStatus(output string) map[string]string {
	info := make(map[string]string)
	if strings.Contains(output, "running") {
		info["running"] = "1"
	} else {
		info["running"] = "0"
	}
	return info
}

// GetSystemMetrics returns current system metrics
func (c *Collector) GetSystemMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// CPU
	if percent, err := cpu.Percent(time.Second, false); err == nil && len(percent) > 0 {
		metrics.CPU.UsagePercent = percent[0]
	}

	// Memory
	if v, err := mem.VirtualMemory(); err == nil {
		metrics.Memory.Total = v.Total
		metrics.Memory.Used = v.Used
		metrics.Memory.Free = v.Free
		metrics.Memory.Available = v.Available
		metrics.Memory.UsedPercent = v.UsedPercent
	}

	if s, err := mem.SwapMemory(); err == nil {
		metrics.Memory.SwapTotal = s.Total
		metrics.Memory.SwapUsed = s.Used
		metrics.Memory.SwapPercent = s.UsedPercent
	}

	// Load
	if l, err := load.Avg(); err == nil {
		metrics.Load.Load1 = l.Load1
		metrics.Load.Load5 = l.Load5
		metrics.Load.Load15 = l.Load15
	}

	// Uptime
	if uptime, err := host.Uptime(); err == nil {
		metrics.Uptime = int64(uptime)
	}

	return metrics, nil
}

// GetOverview returns a monitoring overview
func (c *Collector) GetOverview() (*MonitorOverview, error) {
	overview := &MonitorOverview{
		Timestamp: time.Now(),
	}

	// System metrics
	if system, err := c.GetSystemMetrics(); err == nil {
		overview.System = *system
	}

	// Disk metrics
	partitions, _ := disk.Partitions(false)
	for _, p := range partitions {
		if strings.HasPrefix(p.Fstype, "tmpfs") || p.Fstype == "devtmpfs" {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		dm := DiskMetrics{
			Device:      p.Device,
			MountPoint:  p.Mountpoint,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		}

		overview.Disks = append(overview.Disks, dm)
	}

	// Network metrics
	interfaces, _ := net.IOCounters(true)
	for _, iface := range interfaces {
		if iface.Name == "lo" {
			continue
		}

		nm := NetworkMetrics{
			Interface: iface.Name,
			RxBytes:   iface.BytesRecv,
			TxBytes:   iface.BytesSent,
			RxPackets: iface.PacketsRecv,
			TxPackets: iface.PacketsSent,
			RxErrors:  iface.Errin,
			TxErrors:  iface.Errout,
			RxDropped: iface.Dropin,
			TxDropped: iface.Dropout,
		}

		overview.Network = append(overview.Network, nm)
	}

	// Service health
	services := []string{"nosd", "nos-agent", "caddy"}
	for _, service := range services {
		sm := c.getServiceMetrics(service)
		if sm != nil {
			overview.Services = append(overview.Services, *sm)
		}
	}

	return overview, nil
}

// getServiceMetrics gets metrics for a single service
func (c *Collector) getServiceMetrics(service string) *ServiceMetrics {
	cmd := exec.Command("systemctl", "show", service,
		"--property=ActiveState,SubState,MainPID")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	props := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			props[parts[0]] = parts[1]
		}
	}

	sm := &ServiceMetrics{
		Name:     service,
		State:    props["ActiveState"],
		SubState: props["SubState"],
		Active:   props["ActiveState"] == "active",
		Running:  props["SubState"] == "running",
	}

	if pid, err := strconv.Atoi(props["MainPID"]); err == nil && pid > 0 {
		sm.MainPID = pid
	}

	return sm
}

// updateLastValue updates the last known value for a metric
func (c *Collector) updateLastValue(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastValues[key] = value
}

// GetLastValue retrieves the last known value for a metric
func (c *Collector) GetLastValue(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.lastValues[key]
	return val, ok
}

// mergeMaps merges two string maps
func mergeMaps(a, b map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}
