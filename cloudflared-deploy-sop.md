基于你提供的方案，我为你整理了一套**可直接落地执行的SOP（标准操作流程）**，包含前置检查、详细步骤、验证节点和故障回滚机制。

---

# Cloudflare Tunnel 远程开发环境搭建 SOP

**文档版本**：v1.0  
**适用对象**：鸿蒙平板用户 / WebShell 开发环境  
**预计耗时**：15-20 分钟（首次搭建）  
**风险等级**：低（零防火墙改动，可随时回滚）

---

## Phase 1: 前置条件检查（Pre-check）

### 1.1 环境确认清单
| 检查项 | 命令/方法 | 通过标准 |
|-------|----------|---------|
| **服务器架构** | `uname -m` | 输出 `x86_64` 或 `aarch64` |
| **出站网络** | `curl -I https://cloudflare.com` | HTTP 200 响应 |
| **端口占用** | `lsof -i :3000` | 无输出（或确认目标端口可用） |
| **开发服务状态** | `curl http://localhost:3000` | 返回预期页面内容 |
| **域名权限** | 确认拥有 Cloudflare 托管的域名 | 可在 dash.cloudflare.com 看到域名 |

### 1.2 依赖安装（如缺失）
```bash
# 检查并安装基础工具（Ubuntu/Debian）
apt-get update && apt-get install -y curl wget jq

# 检查并安装基础工具（CentOS/RHEL）
yum install -y curl wget jq
```

**检查点 1**：执行 `which curl`，确认输出 `/usr/bin/curl` 则通过。

---

## Phase 2: cloudflared 安装（Installation）

### 2.1 下载对应版本
```bash
# 自动检测架构并下载最新版
ARCH=$(uname -m)
case $ARCH in
  x86_64) CF_URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64" ;;
  aarch64) CF_URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64" ;;
  *) echo "不支持的架构: $ARCH"; exit 1 ;;
esac

wget -q "$CF_URL" -O /usr/local/bin/cloudflared
chmod +x /usr/local/bin/cloudflared
```

### 2.2 验证安装
```bash
cloudflared --version
```

**预期输出**：
```
cloudflared version 2025.x.x (built ...)
```

**检查点 2**：版本号正常显示，无 "command not found" 错误。

---

## Phase 3: 快速验证模式（Quick Mode）

> **适用场景**：首次测试，5分钟快速验证连通性  
> **有效期**：临时域名，重启即失效

### 3.1 启动临时隧道
```bash
# 在独立终端/screen 中执行
cloudflared tunnel --url http://localhost:3000 --metrics localhost:45678
```

**预期输出**：
```
Your quick Tunnel has been created! Visit it at:
https://abc123-def456.trycloudflare.com
```

### 3.2 平板端验证
1. 鸿蒙平板浏览器访问上述 HTTPS 链接
2. 确认能看到开发服务页面

**检查点 3**：平板能正常访问，页面加载完整。

### 3.3 停止临时隧道
在服务器端按 `Ctrl+C` 终止进程。

---

## Phase 4: 生产级固定域名配置（Production Setup）

### 4.1 账号授权
```bash
cloudflared tunnel login
```

**操作步骤**：
1. 复制输出的 URL（如 `https://dash.cloudflare.com/argotunnel?...`）
2. WebShell 中若无法直接点击，使用 `echo "上述URL"` 然后选中复制
3. 浏览器打开链接，选择授权域名（如 `yourdomain.com`）
4. 授权成功后，服务器端会自动生成 `~/.cloudflared/cert.pem`

**检查点 4**：执行 `ls -la ~/.cloudflared/cert.pem`，文件存在且大小 > 0。

### 4.2 创建永久隧道
```bash
# 创建隧道（命名建议：dev-用户名-项目名）
cloudflared tunnel create dev-server

# 记录输出中的 Tunnel ID
# 示例输出：Tunnel credentials written to /root/.cloudflared/8c9b5c5f-...json
# Tunnel ID: 8c9b5c5f-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

**关键操作**：立即记录 Tunnel ID
```bash
TUNNEL_ID="8c9b5c5f-xxxx-xxxx-xxxx-xxxxxxxxxxxx"  # 替换为实际ID
echo "TUNNEL_ID=$TUNNEL_ID" >> ~/.bashrc
```

### 4.3 配置 DNS 路由
```bash
# 绑定子域名（如 dev.yourdomain.com）
cloudflared tunnel route dns dev-server dev.yourdomain.com
```

**验证**：访问 Cloudflare Dashboard → DNS 记录，确认出现 CNAME 记录 `dev` → `8c9b5c5f-xxxx.cfargotunnel.com`

### 4.4 编写配置文件
```bash
mkdir -p ~/.cloudflared

cat > ~/.cloudflared/config.yml << EOF
tunnel: $TUNNEL_ID
credentials-file: /root/.cloudflared/$TUNNEL_ID.json

# 可选：日志配置
logfile: /var/log/cloudflared.log
loglevel: info

# 传输配置（优化连接稳定性）
protocol: auto
retries: 5

ingress:
  - hostname: dev.yourdomain.com
    service: http://localhost:3000
    # 如需 Basic Auth（兜底方案）
    # originRequest:
    #   noTLSVerify: true
  - service: http_status:404
EOF
```

**检查点 5**：执行 `cloudflared tunnel ingress validate ~/.cloudflared/config.yml`，输出 `Validating rules... OK`

---

## Phase 5: 安全加固（Security Hardening）

### 5.1 Cloudflare Access 配置（强烈推荐）

**操作路径**：
1. 访问 https://one.dash.cloudflare.com
2. Access → Applications → Add an application → Self-hosted
3. 配置参数：
   - **Application Name**: `Dev Server`
   - **Domain**: `dev.yourdomain.com`
   - **Session Duration**: `24 hours`（或 `1 month` 个人使用）

4. **Policies** → Add a Policy：
   - Policy Name: `Allow Dev Team`
   - Action: Allow
   - Include: Email `your-email@example.com`（输入你的邮箱）

5. **Identity Providers**: 选择 "One-time PIN"（邮箱验证码，最简单）

**验证**：保存后，浏览器隐身模式访问 `https://dev.yourdomain.com`，应看到 Cloudflare 登录页而非直接访问内容。

### 5.2 服务层 Basic Auth（备用方案）
如果无法使用 Cloudflare Access，在开发服务端添加密码：

**Node.js/Vite 项目**：
```bash
npm install -D basic-auth-connect
```

修改 `vite.config.js`：
```javascript
import basicAuth from 'basic-auth-connect';

export default {
  server: {
    host: '0.0.0.0',
    port: 3000,
    middlewareMode: true,
    setupMiddlewares(middlewares) {
      middlewares.unshift({
        name: 'basic-auth',
        configureServer(server) {
          server.middlewares.use(basicAuth('admin', 'your-strong-password'));
        }
      });
      return middlewares;
    }
  }
};
```

---

## Phase 6: 系统服务化（Systemd Service）

### 6.1 创建服务文件
```bash
cat > /etc/systemd/system/cloudflared-dev.service << 'EOF'
[Unit]
Description=Cloudflare Tunnel for Dev Server
Documentation=https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
Environment="TUNNEL_METRICS=localhost:45678"
Environment="TUNNEL_LOGFILE=/var/log/cloudflared.log"
Environment="TUNNEL_LOGLEVEL=info"

ExecStart=/usr/local/bin/cloudflared tunnel run --config /root/.cloudflared/config.yml dev-server
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
StartLimitInterval=600
StartLimitBurst=3

# 安全加固（可选）
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true

[Install]
WantedBy=multi-user.target
EOF
```

### 6.2 启动并启用
```bash
# 重载配置
systemctl daemon-reload

# 开机自启
systemctl enable cloudflared-dev

# 启动服务
systemctl start cloudflared-dev

# 查看状态（确认 Active: active (running)）
systemctl status cloudflared-dev --no-pager
```

**检查点 6**：状态显示 `active (running)`，无红色错误信息。

### 6.3 日志监控
```bash
# 实时查看日志
journalctl -u cloudflared-dev -f -n 50

# 检查连接状态
cloudflared tunnel info dev-server
```

---

## Phase 7: 鸿蒙平板端配置（Client Side）

### 7.1 浏览器访问
1. **首选浏览器**：华为浏览器 / Edge（Chromium 内核支持较好）
2. **地址栏输入**：`https://dev.yourdomain.com`
3. **登录流程**：
   - 首次访问：输入邮箱 → 收取验证码 → 进入开发页面
   - 勾选"记住此设备"（30天内免登）

### 7.2 添加到桌面（快捷方式）
1. 页面加载完成后，点击浏览器菜单 → "添加到桌面"
2. 鸿蒙系统会生成桌面图标，下次一键直达

### 7.3 开发者工具（调试）
如需查看控制台：
- 华为浏览器：设置 → 网页 → 开发者工具（需开启开发者模式）

---

## Phase 8: 故障排查手册（Troubleshooting）

### 8.1 隧道无法启动
**现象**：`systemctl status` 显示 `failed`

**排查步骤**：
```bash
# 1. 检查配置文件语法
cloudflared tunnel ingress validate ~/.cloudflared/config.yml

# 2. 检查证书是否存在
ls -la ~/.cloudflared/*.json ~/.cloudflared/cert.pem

# 3. 手动前台运行查看详细错误
cloudflared tunnel run dev-server --loglevel debug
```

**常见错误**：
- `credentials file not found`: 检查 `config.yml` 中的路径是否为绝对路径
- `tunnel not found`: Tunnel ID 错误，执行 `cloudflared tunnel list` 核对

### 8.2 平板无法访问（502 Bad Gateway）
**排查链条**：
```bash
# 1. 检查本地服务是否监听正确地址
netstat -tlnp | grep 3000  # 应显示 0.0.0.0:3000 或 :::3000

# 2. 本地测试连通性
curl http://localhost:3000

# 3. 检查隧道状态
cloudflared tunnel info dev-server  # 看 Connections 是否为 2（正常双连接）
```

**解决方案**：若开发服务只监听 `127.0.0.1`，修改为 `0.0.0.0`：
```bash
# Vite
npm run dev -- --host 0.0.0.0

# React
HOST=0.0.0.0 npm start
```

### 8.3 连接间歇性断开
**优化措施**：
```bash
# 编辑 config.yml 增加心跳配置
echo "heartbeat-interval: 10s" >> ~/.cloudflared/config.yml
systemctl restart cloudflared-dev
```

### 8.4 Access 登录循环
**现象**：输入验证码后回到登录页

**解决**：
1. 检查浏览器是否阻止第三方 Cookie（鸿蒙浏览器默认允许，但隐私模式可能阻止）
2. Cloudflare Dashboard → Access → Application → 你的应用 → Settings
3. 关闭 "Instant Auth"（如启用），延长 Session Duration 至 7 days

---

## Phase 9: 回滚与清理（Rollback）

### 9.1 临时停用隧道
```bash
systemctl stop cloudflared-dev
systemctl disable cloudflared-dev
```

### 9.2 完全卸载
```bash
# 1. 停止并删除服务
systemctl stop cloudflared-dev
rm /etc/systemd/system/cloudflared-dev.service
systemctl daemon-reload

# 2. 删除二进制文件
rm /usr/local/bin/cloudflared

# 3. 删除配置（谨慎操作）
rm -rf ~/.cloudflared

# 4. 清理 Cloudflare DNS（手动）
# 登录 Dashboard → DNS → 删除 dev.yourdomain.com 的 CNAME 记录

# 5. 删除 Tunnel（可选）
cloudflared tunnel delete dev-server
```

### 9.3 清理 Access 应用（可选）
Cloudflare Dashboard → Access → Applications → Dev Server → Delete

---

## Phase 10: 维护与监控（Maintenance）

### 10.1 日常检查命令
```bash
# 快速健康检查
cloudflared tunnel info dev-server | grep -E "Connections|Status"

# 查看最近错误
journalctl -u cloudflared-dev --since "1 hour ago" | grep -i error
```

### 10.2 更新 cloudflared
```bash
# 备份配置
cp -r ~/.cloudflared ~/.cloudflared.bak

# 重复 Phase 2 下载步骤（会自动覆盖旧版本）

# 验证后重启
systemctl restart cloudflared-dev
```

### 10.3 多项目扩展（可选）
如需暴露多个端口（如前端 3000 + 后端 8080）：

```yaml
# config.yml 修改 ingress 部分
ingress:
  - hostname: dev.yourdomain.com
    service: http://localhost:3000
  - hostname: api.yourdomain.com
    service: http://localhost:8080
  - service: http_status:404
```

执行：
```bash
cloudflared tunnel route dns dev-server api.yourdomain.com
systemctl restart cloudflared-dev
```

---

## 执行检查表（Checklist）

| 步骤 | 完成确认 | 执行人 | 时间 |
|-----|---------|-------|------|
| Phase 1 前置检查通过 | [ ] | | |
| cloudflared 安装成功 | [ ] | | |
| 临时隧道测试通过 | [ ] | | |
| 固定隧道创建并记录 ID | [ ] | | |
| Cloudflare Access 配置完成 | [ ] | | |
| Systemd 服务运行正常 | [ ] | | |
| 鸿蒙平板成功访问 | [ ] | | |

**审批**：_________  
**日期**：_________

---

这套 SOP 可直接复制到团队 Wiki 或打印为操作手册使用。
