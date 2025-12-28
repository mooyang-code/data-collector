package dnsproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/mooyang-code/data-collector/pkg/config"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
)

// ServerResponse 服务端响应结构
type ServerResponse struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    []ServerDNSRecord `json:"data"`
	Total   int               `json:"total"`
}

// ServerDNSRecord 服务端 DNS 记录
type ServerDNSRecord struct {
	Domain    string    `json:"domain"`
	BestIPs   string    `json:"best_ips"` // "1.2.3.4+5.6.7.8"
	ResolveAt time.Time `json:"resolve_at"`
	Success   bool      `json:"success"`
}

// ScheduledFetchDNS 定时器入口函数 - 定时获取 DNS 记录
func ScheduledFetchDNS(c context.Context, _ string) error {
	ctx := trpc.CloneContext(c)
	nodeID, version := config.GetNodeInfo()
	log.WithContextFields(ctx, "func", "ScheduledFetchDNS", "version", version, "nodeID", nodeID)

	log.DebugContext(ctx, "ScheduledFetchDNS Enter")
	if err := FetchDNSRecords(ctx); err != nil {
		log.ErrorContextf(ctx, "scheduled fetch DNS failed: %v", err)
		return err
	}
	log.DebugContext(ctx, "ScheduledFetchDNS Success")
	return nil
}

// FetchDNSRecords 获取 DNS 记录
func FetchDNSRecords(ctx context.Context) error {
	// 1. 获取服务端地址
	serverIP, serverPort := config.GetServerInfo()
	if serverIP == "" {
		log.DebugContext(ctx, "no server IP configured, skipping DNS fetch")
		return nil
	}

	// 2. 构建请求 URL
	url := fmt.Sprintf("http://%s:%d/gateway/dnsproxy/GetDNSRecordList", serverIP, serverPort)

	// 3. 发送 HTTP 请求
	respData, err := fetchFromServer(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to fetch DNS records: %w", err)
	}

	// 4. 解析响应
	serverRecords, err := parseServerResponse(respData)
	if err != nil {
		return fmt.Errorf("failed to parse DNS response: %w", err)
	}

	log.DebugContextf(ctx, "Fetched %d DNS records from server", len(serverRecords))

	// 5. 对每个域名的 IP 列表进行探测和排序，转换为内部格式
	records := make([]*DNSRecord, 0, len(serverRecords))
	for _, srvRecord := range serverRecords {
		// 从 best_ips 中解析 IP 列表
		ips := parseBestIPs(srvRecord.BestIPs)
		log.DebugContextf(ctx, "Domain %s has %d IPs to probe", srvRecord.Domain, len(ips))

		// 调用 probeAndSort（传递域名，内部会查找探测配置）
		ipList := probeAndSort(ctx, srvRecord.Domain, ips)

		// 记录可用 IP 数量
		availableCount := 0
		for _, ip := range ipList {
			if ip.Available {
				availableCount++
			}
		}
		log.DebugContextf(ctx, "Domain %s: %d/%d IPs available", srvRecord.Domain, availableCount, len(ips))

		// 创建 DNSRecord
		record := &DNSRecord{
			Domain:    srvRecord.Domain,
			IPList:    ipList,
			ResolveAt: srvRecord.ResolveAt,
			Success:   srvRecord.Success,
		}
		records = append(records, record)
	}

	// 6. 更新全局变量
	updateDNSRecords(records)

	log.DebugContextf(ctx, "DNS records updated successfully, total: %d", len(records))
	return nil
}

// fetchFromServer 从服务端获取数据
func fetchFromServer(ctx context.Context, url string) ([]byte, error) {
	httpClient := &http.Client{Timeout: 5 * time.Second}

	var respData []byte
	err := retry.Do(
		func() error {
			return sendSingleRequest(ctx, url, httpClient, &respData)
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			log.WarnContextf(ctx, "retrying DNS fetch request, attempt: %d, error: %v", n+1, err)
		}),
		retry.Context(ctx),
	)
	return respData, err
}

// sendSingleRequest 发送单次 HTTP 请求
func sendSingleRequest(ctx context.Context, url string, httpClient *http.Client, respData *[]byte) error {
	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to create DNS fetch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyData, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DNS fetch request failed with status: %d, response: %s", resp.StatusCode, string(bodyData))
	}

	// 读取响应
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	*respData = data
	return nil
}

// parseServerResponse 解析服务端响应
func parseServerResponse(respData []byte) ([]ServerDNSRecord, error) {
	var serverResp ServerResponse
	if err := json.Unmarshal(respData, &serverResp); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}

	// 检查响应状态码
	if serverResp.Code != 200 {
		return nil, fmt.Errorf("server returned error code: %d, message: %s", serverResp.Code, serverResp.Message)
	}

	return serverResp.Data, nil
}

// parseBestIPs 解析 best_ips 字符串为 IP 列表
// 格式: "1.2.3.4+5.6.7.8+9.10.11.12"
func parseBestIPs(bestIPs string) []string {
	if bestIPs == "" {
		return nil
	}

	parts := strings.Split(bestIPs, "+")
	ips := make([]string, 0, len(parts))
	for _, ip := range parts {
		if trimmed := strings.TrimSpace(ip); trimmed != "" {
			ips = append(ips, trimmed)
		}
	}
	return ips
}
