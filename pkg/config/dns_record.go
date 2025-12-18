package config

import "time"

const TBDNSRecord = "dns_record"

// IPInfo 单个 IP 的信息
type IPInfo struct {
	IP        string    `json:"ip"`        // IP 地址
	Latency   int64     `json:"latency"`   // 延迟（毫秒）
	Available bool      `json:"available"` // 是否可用
	LastPing  time.Time `json:"last_ping"` // 最后探测时间
}

// DNSRecord DNS 解析记录
type DNSRecord struct {
	Domain    string    `json:"domain"`     // 域名
	IPList    []IPInfo  `json:"ip_list"`    // IP 列表
	ResolveAt time.Time `json:"resolve_at"` // 解析时间
	Success   bool      `json:"success"`    // 解析是否成功

	// AccessUrl 访问该表的接口 URL
	AccessUrl string `json:"-"`
}

// SchemaID 实现接口 APICacher
func (DNSRecord) SchemaID() string {
	return TBDNSRecord
}

// URL 实现接口 APICacher
func (d DNSRecord) URL() string {
	return d.AccessUrl
}

// SearchFields 实现接口 APICacher
func (DNSRecord) SearchFields() map[string]string {
	return map[string]string{TBDNSRecord: "Domain"}
}

// FilterKey 实现接口 APICacher
func (DNSRecord) FilterKey() string {
	return ""
}

// GetAllDNSRecords 获取所有 DNS 记录
func GetAllDNSRecords() []*DNSRecord {
	records, ok := GetAll(TBDNSRecord).([]*DNSRecord)
	if !ok {
		return nil
	}
	return records
}

// GetDNSRecordByDomain 根据域名获取 DNS 记录
func GetDNSRecordByDomain(domain string) *DNSRecord {
	if domain == "" {
		return nil
	}

	records := GetAllDNSRecords()
	if records == nil {
		return nil
	}

	for _, record := range records {
		if record.Domain == domain {
			return record
		}
	}
	return nil
}

// GetBestIP 获取指定域名的最优 IP（延迟最低且可用）
func GetBestIP(domain string) string {
	record := GetDNSRecordByDomain(domain)
	if record == nil || len(record.IPList) == 0 {
		return ""
	}

	var bestIP string
	var minLatency int64 = -1

	for _, ip := range record.IPList {
		// 只选择可用的 IP
		if !ip.Available {
			continue
		}

		// 选择延迟最低的
		if minLatency < 0 || ip.Latency < minLatency {
			minLatency = ip.Latency
			bestIP = ip.IP
		}
	}

	return bestIP
}

// GetAvailableIPs 获取指定域名所有可用的 IP 列表
func GetAvailableIPs(domain string) []string {
	record := GetDNSRecordByDomain(domain)
	if record == nil || len(record.IPList) == 0 {
		return nil
	}

	var ips []string
	for _, ip := range record.IPList {
		if ip.Available {
			ips = append(ips, ip.IP)
		}
	}
	return ips
}
