package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ProxyValidator 用于验证代理的结构体
type ProxyValidator struct {
	proxy    string
	response string
	success  bool
}

// normalizeIP 用于处理返回的 IP，去掉多余字符
func normalizeIP(raw string) string {
	// 去除不可见字符、空格和末尾多余符号
	return strings.TrimSpace(strings.ReplaceAll(raw, "\n", ""))
}

// validateProxy 验证代理 IP 是否可用
func validateProxy(proxy string) (bool, string) {
	proxyURL, err := url.Parse(proxy) // 直接解析代理 URL
	if err != nil {
		return false, ""
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "http://ifconfig.me", nil)
	if err != nil {
		return false, ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, ""
	}

	// 处理返回的 IP 地址
	responseIP := normalizeIP(string(body))
	proxyIP := strings.Split(proxyURL.Host, ":")[0] // 提取代理 IP 部分

	return responseIP == proxyIP, responseIP
}

// worker 负责处理代理验证并实时写入文件
func worker(jobs <-chan string, successFile *os.File, successFileLock *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	for proxy := range jobs {
		success, response := validateProxy(proxy)
		if success {
			// 实时写入成功代理到文件
			successFileLock.Lock()
			successFile.WriteString(proxy + "\n")
			successFileLock.Unlock()

			fmt.Printf("✅ 成功代理: %s\n", proxy)
		} else {
			fmt.Printf("❌ 无效代理: %s 返回值: %s\n", proxy, response)
		}
	}
}

func main() {
	inputFile := "res.txt"
	outputFile := "success.txt"

	jobs := make(chan string, 100)
	var wg sync.WaitGroup
	var successFileLock sync.Mutex

	// 打开文件以实时写入成功代理
	successFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("无法创建文件: %v\n", err)
		return
	}
	defer successFile.Close()

	// 默认 10 个线程
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(jobs, successFile, &successFileLock, &wg)
	}

	// 读取代理列表
	go func() {
		file, err := os.Open(inputFile)
		if err != nil {
			fmt.Printf("无法打开文件: %v\n", err)
			close(jobs)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			proxy := scanner.Text()
			jobs <- proxy
		}
		close(jobs)
	}()

	// 等待所有工作完成
	wg.Wait()
	fmt.Println("所有代理验证完成！")
}
