# TunMan 前置准备

> 使用 TunMan 前，需要完成以下一次性准备工作。

---

## 1. 域名

购买域名（约 10-100 元/年）：
- 国内：阿里云、腾讯云（需实名）
- 国外：Namecheap

---

## 2. Cloudflare 账号

注册：https://dash.cloudflare.com/sign-up

免费版完全够用。

---

## 3. 域名接入 Cloudflare

1. 添加域名到 Cloudflare Dashboard
2. 修改域名 NS 记录为 Cloudflare 提供的地址
3. 等待生效（Active 状态）

---

## 4. 安装 cloudflared

```bash
# Linux AMD64
curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# 或其他方式：https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/
```

---

## 5. 创建 Tunnel

```bash
# 登录授权
cloudflared tunnel login

# 创建隧道
cloudflared tunnel create my-tunnel
# 记录输出的 Tunnel ID

# 配置 DNS
cloudflared tunnel route dns my-tunnel dev.example.com
```

---

## 6. 记录信息

创建完成后，需要这些信息：

| 字段 | 示例 |
|------|------|
| Tunnel ID | `3c1fa5c8-xxxx-xxxx` |
| 子域名 | `dev.example.com` |
| 本地端口 | `3000` |

---

**完成后，使用 `tunman` 管理你的隧道。**
