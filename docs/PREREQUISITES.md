# TunMan 前置准备

本文档只保留“使用 TunMan 前必须完成什么”。

如果你已经具备这些条件，可以继续看 `docs/USAGE.md`。

## 1. 先理解三种隧道类型

TunMan 目前支持三种隧道：

- `Custom`：固定域名，适合长期使用
- `Testing`：自动生成测试前缀，适合临时调试
- `Ephemeral`：快速临时隧道，适合本地联调

不同类型对前置条件的要求不同：

| 类型 | 是否需要 `cloudflared` | 是否需要 Cloudflare 域名 | 是否需要登录 Cloudflare |
|------|------------------------|---------------------------|---------------------------|
| `Custom` | 是 | 是 | 是 |
| `Testing` | 是 | 是 | 是 |
| `Ephemeral` | 否 | 否 | 否 |

## 2. 必装软件

### 2.1 安装 `screen`

TunMan 通过 `screen` 在后台运行隧道进程，因此这是所有模式的基础依赖。

常见检查方式：

```bash
screen --version
```

### 2.2 安装 `cloudflared`

只有 `Custom` 和 `Testing` 隧道需要 `cloudflared`。

Linux AMD64 示例：

```bash
curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb
```

其他平台或架构请参考官方文档：
`https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/`

安装后检查：

```bash
cloudflared --version
```

## 3. Cloudflare 侧准备

如果你只使用 `Ephemeral` 隧道，可以跳过这一节。

### 3.1 准备 Cloudflare 账号

先注册并登录 Cloudflare：
`https://dash.cloudflare.com/sign-up`

### 3.2 准备可用域名

你需要一个已经由 Cloudflare 托管的域名。

基本要求：

- 域名已经添加到 Cloudflare
- 域名 NS 已切换到 Cloudflare
- Cloudflare Dashboard 中状态为可用

TunMan 在创建 `Custom` / `Testing` 隧道时，会基于你填写的根域名自动配置 tunnel 和 DNS 路由。

### 3.3 完成 `cloudflared` 登录授权

```bash
cloudflared tunnel login
```

完成后，本机会生成 Cloudflare 授权文件。TunMan 会在启动时检查是否已完成登录。

## 4. 本地服务准备

无论使用哪种隧道类型，你都需要先有一个本地服务端口可供转发，例如：

- `3000`
- `5173`
- `8080`

确认服务已经在本机正常运行后，再使用 TunMan 创建隧道。

## 5. 使用前自检

建议在首次使用前手动确认：

### 5.1 所有模式都建议检查

```bash
screen --version
```

### 5.2 如果你要使用 `Custom` 或 `Testing`

再额外确认：

```bash
cloudflared --version
cloudflared tunnel login
```

如果你打算创建固定域名隧道，还要确认：

- 根域名已经接入 Cloudflare
- 你知道要使用的根域名，例如 `example.com`
- 你知道本地服务端口，例如 `3000`

## 6. TunMan 会帮你做什么

对于 `Custom` / `Testing` 隧道，TunMan 会自动处理这些步骤：

- 创建 Cloudflare Tunnel
- 自动生成或填写子域名前缀
- 配置 DNS 路由
- 生成 `cloudflared` 运行配置
- 在后台启动隧道进程

你不需要手工执行 `cloudflared tunnel create`、`route dns`、写 `config.yml` 这些步骤。

## 7. 下一步

前置准备完成后，继续阅读：

- `docs/USAGE.md`：TunMan 使用教程
