# TunMan 使用操作文档

本文档面向日常使用者，介绍 `tunman` 的安装、启动和常见操作流程。

## 1. 前置准备

首次使用前请完成：

- 已安装 `screen`
- 如需使用 `Custom` / `Testing` 隧道：已安装并登录 `cloudflared`
- 如需使用自定义域名：已准备可用根域名

详细步骤见 `docs/PREREQUISITES.md`。

## 2. 安装与启动

```bash
# 在项目根目录构建
go build -o tunman ./cmd/tunman

# 启动 TUI（默认）
./tunman
```

也可安装到系统路径：

```bash
sudo mv tunman /usr/local/bin/
tunman
```

## 3. 运行模式

### 3.1 TUI 模式（推荐）

直接运行 `tunman` 进入终端界面，支持列表管理、创建、日志查看等操作。

### 3.2 CLI 模式

```bash
tunman version
```

说明：当前 CLI 入口已存在，但 `list`、`status`、`start`、`stop` 仍在完善中，完整体验建议使用 TUI。

## 4. TUI 页面与快捷键

### 4.1 全局快捷键

- `Ctrl+R`：手动刷新状态
- `?`：打开帮助页
- `Esc` / `b`：返回上一页
- `q`：退出程序（会自动清理临时隧道）

### 4.2 List 页面

- `j/k` 或 `↑/↓`：上下选择
- `/`：进入搜索模式（按 `Enter` 或 `Esc` 退出）
- `f`：切换分类过滤（All/Custom/Testing/Ephemeral）
- `s/x/r`：启动/停止/重启选中隧道
- `l` 或 `Enter`：打开日志页
- `a`：打开创建页
- `y`：复制当前隧道 URL
- `d`：删除（`y` 确认，`n` 或 `Esc` 取消）
- `g/G`：跳到首条/末条

### 4.3 Create 页面

- `j/k` 或 `Tab`：选择字段
- `Enter` 或 `Space`：进入字段编辑、切换分类或提交创建
- `←/→`：切换隧道分类
- `Ctrl+S`：提交创建

字段说明：

- `Custom`：填写 `Base Domain`、`Subdomain`、`Port`，可选 `Description`
- `Testing`：填写 `Base Domain`、`Port`，子域名前缀自动生成
- `Ephemeral`：填写 `Port`，可选 `Description`
- `Tunnel ID` 由程序调用 `cloudflared` 自动创建，不需要手工填写

### 4.4 Logs 页面

- `j/k`：滚动日志
- `PgUp/PgDn`：按半屏滚动
- `g/G`：跳到首行/末行
- `t`：切换跟随模式（Follow ON/OFF）

## 5. 常见操作流程

### 5.1 新建并启动隧道

1. 进入 `Create` 页面填写字段并提交
2. `Custom` / `Testing` 隧道会在创建完成后尝试自动启动
3. 返回 `List` 页面确认状态
4. 按 `l` 查看日志确认运行状态

### 5.2 删除隧道

1. 在 `List` 选中目标隧道
2. 按 `d`
3. 按 `y` 确认删除

## 6. 数据文件位置

TunMan 在用户目录维护运行数据：

```text
~/.tunman/
├── config.json
├── tunnels.json
└── logs/
```

## 7. 故障排查

- 启动失败：检查 `screen` 是否已安装并在 `PATH` 中；`Custom` / `Testing` 还需确认 `cloudflared` 已安装并登录
- 无法启动固定隧道：确认 `Tunnel ID`、DNS 路由及 Cloudflare 权限
- 日志为空：先确认隧道是否已启动，再查看 `~/.tunman/logs/*.log`
- 复制 URL 失败：通常是系统剪贴板工具不可用，不影响隧道运行
