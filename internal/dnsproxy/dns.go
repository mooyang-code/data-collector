package dnsproxy

import (
	"sync"
	"time"
)

// IPInfo 单个 IP 的信息
type IPInfo struct {
	IP        string    `json:"ip"`        // IP 地址
	Latency   int64     `json:"latency"`   // 延迟（微秒）
	Available bool      `json:"available"` // 是否可用
	LastPing  time.Time `json:"last_ping"` // 最后探测时间
}

// DNSRecord DNS 解析记录
type DNSRecord struct {
	Domain    string     `json:"domain"`     // 域名
	IPList    []*IPInfo  `json:"ip_list"`    // IP 列表（已排序：可用的优先，延迟低的优先）
	ResolveAt time.Time  `json:"resolve_at"` // 解析时间
	Success   bool       `json:"success"`    // 解析是否成功
}

// 全局变量（用 sync.RWMutex 保护）
var (
	dnsRecords map[string]*DNSRecord
	dnsMutex   sync.RWMutex
)

// Init 初始化 DNS 代理
func Init() {
	dnsMutex.Lock()
	defer dnsMutex.Unlock()
	dnsRecords = make(map[string]*DNSRecord)
}

// GetBestIP 获取指定域名的最优 IP（第一个可用 IP）
func GetBestIP(domain string) string {
	dnsMutex.RLock()
	defer dnsMutex.RUnlock()

	record, exists := dnsRecords[domain]
	if !exists || record == nil || len(record.IPList) == 0 {
		return ""
	}

	// 返回第一个可用的 IP
	for _, ipInfo := range record.IPList {
		if ipInfo.Available {
			return ipInfo.IP
		}
	}
	return ""
}

// GetAvailableIPs 获取指定域名所有可用的 IP 列表
func GetAvailableIPs(domain string) []string {
	dnsMutex.RLock()
	defer dnsMutex.RUnlock()

	record, exists := dnsRecords[domain]
	if !exists || record == nil || len(record.IPList) == 0 {
		return nil
	}

	var ips []string
	for _, ipInfo := range record.IPList {
		if ipInfo.Available {
			ips = append(ips, ipInfo.IP)
		}
	}
	return ips
}

// GetDNSRecord 获取指定域名的完整 DNS 记录
func GetDNSRecord(domain string) *DNSRecord {
	dnsMutex.RLock()
	defer dnsMutex.RUnlock()

	record, exists := dnsRecords[domain]
	if !exists {
		return nil
	}
	return record
}

// updateDNSRecords 更新全局 DNS 记录
func updateDNSRecords(records []*DNSRecord) {
	dnsMutex.Lock()
	defer dnsMutex.Unlock()

	// 清空旧记录
	dnsRecords = make(map[string]*DNSRecord)

	// 添加新记录
	for _, record := range records {
		dnsRecords[record.Domain] = record
	}
}
