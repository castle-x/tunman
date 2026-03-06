# TUI 优化方案 — 对齐 tui-design v1.2 规范

日期：2026-02-24

## 第一轮：UI 规范对齐（已完成）

P0-P3 共 8 项优化，已全部完成。

## 第二轮：文本精简（已完成）

- j/k → ↑/↓
- 去除 LV1 术语
- 去除重复统计信息（info bar + footer）
- 去除页面内重复标题
- 创建页面从三步向导改为单页平铺表单

## 第三轮：模型重构 + Controller 自动化

### 隧道分类重新设计

| 旧分类 | 新分类 | 内部值 | 用户输入 | 自动化 |
|--------|--------|--------|---------|--------|
| Protected | 开放域名 | `custom` | 根域名 + 前缀 + 端口 | TunMan 创建 tunnel + DNS |
| ZeroTrust | _(删除)_ | - | - | - |
| - | 测试域名 | `testing` | 根域名 + 端口 | TunMan 生成 UUID 前缀 |
| Ephemeral | 临时隧道 | `quick` | 端口 | cloudflared 随机域名 |

### 模型变更

- 删除：`AccessType`, `AccessValue`, `CategoryZeroTrust`
- 新增：`BaseDomain`（根域名）, `Prefix`（三级前缀）
- 保留：`TunnelID` 作为内部自动管理字段
- `Subdomain` 改为计算方法 `FullDomain()` = `Prefix.BaseDomain`

### Controller 自动化

cloudflared 封装模块，负责执行系统命令：

1. `CheckInstalled()` — 检查 cloudflared 是否安装
2. `CheckLogin()` — 检查 cert.pem 是否存在
3. `SetupTunnel(tunnel)` — 创建 tunnel + 配置 DNS + 生成 config
4. `TeardownTunnel(tunnel)` — 删除 tunnel + 清理 DNS
5. `Start(tunnel)` / `Stop(tunnel)` — 启动/停止（适配新分类）

### 实现顺序

1. model/tunnel.go — 字段增删改
2. core/controller.go — cloudflared 自动化
3. ui/create.go — 表单适配新字段
4. ui/ 其他文件 — 分类引用更新
5. i18n — key 更新

## 第四轮：cloudflared 环境预检

TUI 启动时检查 cloudflared 环境，结果缓存在 Model 中：

- `cfInstalled bool` — cloudflared 二进制是否存在
- `cfLoggedIn bool` — cert.pem 是否存在
- info bar 持续显示警告（未安装 / 未登录）
- 创建 Custom/Testing 时 validate 阶段拦截，给中文友好提示
- i18n 新增 `cf_not_installed` / `cf_not_logged_in` / `create_err_cf_*` key

---

# 修复：删除隧道后重建同域名报 DNS 冲突

## 问题

用 tunman 删除隧道后，再用相同域名创建新隧道时报错：
```
dns route failed: Failed to add route: code: 1003, reason: Failed to create record ...
An A, AAAA, or CNAME record with that host already exists.
```

## 原因

- cloudflared CLI **没有**可靠的删除 DNS 记录命令（`--remove` 参数实际不存在或不生效）
- 删除隧道时 CNAME 记录留在 Cloudflare，重建同域名隧道时 `route dns` 默认"新建"而非"覆盖"，于是冲突

## 方案

在 `SetupTunnel` 的 `cloudflared tunnel route dns` 调用中加上 `--overwrite-dns` 参数：
- 若 DNS 记录不存在 → 正常创建
- 若 DNS 记录已存在 → 覆盖为新隧道的 CNAME，不报错

## 改动范围

| 文件 | 改动 |
|------|------|
| `internal/core/controller.go` | `SetupTunnel` 中 `route dns` 命令加 `--overwrite-dns` |
