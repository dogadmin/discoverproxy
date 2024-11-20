import requests as req
from urllib.parse import urlparse
from concurrent.futures import ThreadPoolExecutor

class ProxyVerifier:
    def __init__(self, input_file, output_file, max_workers=10):
        self.input_file = input_file
        self.output_file = output_file
        self.max_workers = max_workers

    def get_true_ip(self):
        try:
            res_json = req.get('http://httpbin.org/ip', timeout=10).json()
            return res_json.get('origin', '')
        except req.RequestException as e:
            print(f"获取真实 IP 失败: {e}")
            return ''

    def parse_proxy(self, proxy_url):
        """
        从http://格式提取代理地址，转换为socks5格式。
        """
        parsed = urlparse(proxy_url)
        if parsed.scheme in ["http", "https"]:  # 仅处理http/https前缀的代理
            return f"socks5://{parsed.hostname}:{parsed.port}"
        return None

    def verify_proxy(self, proxy_url):
        try:
            # 转换为 SOCKS5 格式
            socks5_proxy = self.parse_proxy(proxy_url)
            if not socks5_proxy:
                print(f"无效代理格式: {proxy_url}")
                return

            proxies = {
                'http': socks5_proxy,
                'https': socks5_proxy
            }

            # 验证代理
            res_proxy = req.get('http://httpbin.org/ip', proxies=proxies, timeout=10).json()
            if res_proxy.get('origin') and res_proxy['origin'] != self.true_ip:
                print(f"代理有效: {socks5_proxy}, 代理 IP: {res_proxy['origin']}")
                # 实时写入成功结果
                with open(self.output_file, "a") as file:
                    file.write(f"{socks5_proxy} {res_proxy['origin']}\n")
            else:
                print(f"代理无效: {socks5_proxy}")
        except req.RequestException as e:
            print(f"代理验证失败: {proxy_url}, 错误: {e}")

    def run(self):
        self.true_ip = self.get_true_ip()
        if not self.true_ip:
            print("无法获取真实 IP，停止验证。")
            return

        # 读取代理列表
        with open(self.input_file, 'r') as file:
            proxy_urls = [line.strip() for line in file if line.strip()]

        print(f"读取到 {len(proxy_urls)} 个代理，开始验证...")

        # 多线程验证代理
        with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
            executor.map(self.verify_proxy, proxy_urls)

        print("验证完成！")

# 主函数
if __name__ == "__main__":
    input_file = "1.txt"         # 输入文件，包含代理列表
    output_file = "success"      # 输出文件，存储有效代理
    max_workers = 10             # 最大线程数

    verifier = ProxyVerifier(input_file, output_file, max_workers)
    verifier.run()
