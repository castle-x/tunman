# Cloudflare Tunnel 远程开发指南

> 适用于：鸿蒙平板 / 无 SSH 客户端 / WebShell 环境访问远程开发服务

---

## 方案概述

**Cloudflare Tunnel**（通过 `cloudflared` 客户端实现）是一种安全的内网穿透方案，可将本地开发服务器（如 `localhost:3000`）暴露为公网 HTTPS 链接。

### 核心优势

| 特性 | 说明 |
|------|------|
| **零防火墙配置** | 无需开放任何入站端口（outbound 连接） |
| **HTTPS 自动** | 自动生成 SSL 证书，无需手动配置 |
| **身份验证** | 支持邮箱验证、OTP 等多重登录保护 |
| **完全免费** | 个人使用无流量/带宽限制 |
| **穿透 NAT** | 适用于内网、有云主机无公网 IP 等场景 |

### 原理简图

```
平板浏览器 → HTTPS → Cloudflare CDN → 长连接 → cloudflared → 本地服务:3000
                              ↑
                        （服务器主动连接出去，无需开放端口）
```

---

## 适用场景

- 鸿蒙平板无原生 SSH 客户端，只能通过 WebShell（如腾讯云 OrcaTerm）连接服务器
- 需要临时访问开发服务器启动的页面（Vue/React/Vite 等）
- 不希望开放服务器防火墙端口，要求安全可控
- 需要 HTTPS 链接测试微信/企业微信等受限环境

---

## 安装与配置

### 1. 安装 cloudflared

```bash
# Linux AMD64（大多数云服务器）
wget -q https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -O /usr/local/bin/cloudflared
chmod +x /usr/local/bin/cloudflared

# 验证安装
cloudflared --version
```

> 其他架构（ARM 等）请参考：https://github.com/cloudflare/cloudflared/releases

---

### 2. 快速启动（临时使用）

```bash
# 最简单的用法，直接暴露本地端口
cloudflared tunnel --url http://localhost:3000
```

输出示例：
```
Your quick Tunnel has been created! Visit it at:
https://abc123-def456.trycloudflare.com
```

**特点：**
- 域名随机生成，**每次重启变化**
- 适合一次性调试
- Ctrl+C 即关闭，无残留

---

### 3. 固定域名（推荐）

如需固定链接（方便收藏/分享给协作者）：

```bash
# 1. 登录授权（复制输出的 URL 到浏览器完成授权）
cloudflared tunnel login

# 2. 创建 tunnel（名称可自定义，如 dev-server）
cloudflared tunnel create dev-server

# 3. 记录输出中的 tunnel ID（如：8c9b5c5f-xxx）

# 4. 配置 DNS 路由（需提前在 Cloudflare 添加你的域名）
cloudflared tunnel route dns dev-server dev.yourdomain.com

# 5. 编写配置文件
mkdir -p ~/.cloudflared
cat > ~/.cloudflared/config.yml << 'EOF'
tunnel: <你的-tunnel-ID>
credentials-file: /root/.cloudflared/<你的-tunnel-ID>.json

ingress:
  - hostname: dev.yourdomain.com
    service: http://localhost:3000
  - service: http_status:404
EOF

# 6. 启动
cd ~/.cloudflared && cloudflared tunnel run dev-server
```

---

## 安全加固（必须）

**临时公网地址存在被扫描的风险，强烈建议启用身份验证。**

### 方案 A：Cloudflare Access（推荐）

在 Cloudflare Zero Trust 控制台配置：

1. 访问 https://one.dash.cloudflare.com
2. 进入 **Access → Applications**
3. 点击 **Add an application → Self-hosted**
4. 填写信息：
   - Application name: `Dev Server`
   - Session duration: `24 hours`（按需）
   - Domain: `dev.yourdomain.com`
5. **Policies** 添加规则：
   - 选择登录方式：Email（你的邮箱）
   - 或 OTP（每次输入邮箱收验证码）
6. 保存后，访问链接会先跳转到登录页

> 免费版支持最多 50 个用户，个人开发完全够用。

---

### 方案 B：HTTP Basic Auth（快速）

若使用随机域名且无 Cloudflare Access，可在本地服务层加密码：

**Node.js (Express) 示例：**
```javascript
const basicAuth = require('express-basic-auth');

app.use(basicAuth({
    users: { 'admin': 'your-strong-password' },
    challenge: true,
    realm: 'Dev Server'
}));
```

**Python (Flask) 示例：**
```python
from flask import Flask
from flask_httpauth import HTTPBasicAuth

auth = HTTPBasicAuth()
users = {"admin": "your-strong-password"}

@auth.get_password
def get_pw(username):
    return users.get(username)

@app.route('/')
@auth.login_required
def index():
    return "Protected"
```

---

## 常用命令

| 命令 | 说明 |
|------|------|
| `cloudflared tunnel --url http://localhost:3000` | 快速临时隧道 |
| `cloudflared tunnel list` | 查看所有 tunnel |
| `cloudflared tunnel delete <name>` | 删除 tunnel |
| `cloudflared tunnel info <name>` | 查看 tunnel 状态 |
| `cloudflared tunnel run <name>` | 运行指定 tunnel（前台）|
| `cloudflared service install` | 安装为系统服务 |

---

## 后台运行方案

### 方案 1：tmux/screen（简单）
```bash
tmux new -s tunnel
cloudflared tunnel --url http://localhost:3000
# Ctrl+B 然后按 D 分离会话
tmux attach -t tunnel  # 重新连接
```

### 方案 2：systemd 服务（长期）
```bash
# 1. 创建服务文件
sudo cat > /etc/systemd/system/cloudflared-dev.service << 'EOF'
[Unit]
Description=Cloudflare Tunnel for Dev Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/root/.cloudflared
ExecStart=/usr/local/bin/cloudflared tunnel run dev-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 2. 启动并启用开机自启
sudo systemctl daemon-reload
sudo systemctl enable cloudflared-dev
sudo systemctl start cloudflared-dev

# 3. 查看状态
sudo systemctl status cloudflared-dev
```

---

## 与其他方案对比

| 方案 | 防火墙配置 | HTTPS | 身份验证 | 免费额度 | 适用场景 |
|------|-----------|-------|---------|---------|---------|
| **Cloudflare Tunnel** | 无需 | ✅ 自动 | ✅ 完善 | 无限 | **首推，综合最优** |
| Ngrok | 无需 | ✅ | ⚠️ 简单 | 有限流量 | 快速验证 |
| Tailscale | 无需 | ❌ | ✅ 设备认证 | 20设备免费 | 多设备组网 |
| Nginx 反向代理 | 需要 | 需手动配置 | 需自行实现 | - | 有域名+固定服务 |
| 直接暴露端口 | 需要 | 需手动配置 | ❌ | - | 不推荐，风险高 |

---

## 常见问题

**Q: 免费版有什么限制？**
- 随机域名（可绑定自有域名免费解决）
- Access 身份验证最多 50 用户（个人完全够用）
- 无 SLA 保证（偶尔波动可接受）

**Q: 服务器在内网/无公网 IP 可以用吗？**
- 可以。cloudflared 主动连接 Cloudflare，不依赖入站连接。

**Q: 鸿蒙平板能用吗？**
- 可以。只需要在平板的浏览器中打开生成的 HTTPS 链接即可。

**Q: 服务断了怎么办？**
- cloudflared 会自动重连，建议配合 systemd 或 tmux 使用。

**Q: 如何查看日志？**
```bash
# 前台运行查看实时日志
cloudflared tunnel --url http://localhost:3000 --loglevel debug

# systemd 查看日志
journalctl -u cloudflared-dev -f
```

---

## 参考链接

- Cloudflare Tunnel 文档：https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/
- GitHub Releases：https://github.com/cloudflare/cloudflared/releases
- Cloudflare Zero Trust 控制台：https://one.dash.cloudflare.com

---

*文档版本：2026-02-05*
