package eyeduxsdk

import (
	"net"
	"os"
	"runtime"
	"sort"
)

func isRuntimeMetadataEvent(eventType EventEyeduxType) bool {
	return eventType == EventEyeduxTypeAudit ||
		eventType == EventEyeduxTypeMetric ||
		(len(eventType) >= len("system-") && eventType[:len("system-")] == "system-")
}

func collectRuntimeMetadata() map[string]any {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	metadata := map[string]any{
		"runtime": map[string]any{
			"os":           runtime.GOOS,
			"architecture": runtime.GOARCH,
			"go_version":   runtime.Version(),
			"cpu_count":    runtime.NumCPU(),
			"goroutines":   runtime.NumGoroutine(),
		},
		"memory": map[string]any{
			"alloc_bytes":      memory.Alloc,
			"heap_alloc_bytes": memory.HeapAlloc,
			"sys_bytes":        memory.Sys,
		},
	}

	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		metadata["hostname"] = hostname
	}
	metadata["ip_addresses"] = localIPAddresses()

	return metadata
}

func localIPAddresses() []string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return []string{}
	}

	ips := make([]string, 0, len(addresses))
	for _, address := range addresses {
		network, ok := address.(*net.IPNet)
		if !ok || network.IP.IsLoopback() || network.IP.IsUnspecified() || network.IP.IsLinkLocalUnicast() {
			continue
		}
		ips = append(ips, network.IP.String())
	}
	sort.Strings(ips)
	return ips
}
