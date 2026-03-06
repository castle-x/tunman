# TunMan

> 基于 Go + Bubble Tea 的 Cloudflare Tunnel 管理工具，默认提供交互式 TUI。

## 概览

TunMan 用来集中管理本地 Cloudflare Tunnel 配置、运行状态和日志，适合经常在终端里启动、停止、查看和清理隧道的场景。

当前项目以 TUI 体验为主；CLI 子命令仍在逐步补全。

## 当前能力

- 交互式 TUI：`List / Create / Delete / Logs / Help`
- 自动刷新隧道状态
- 按分类过滤与关键字搜索
- 启动 / 停止 / 重启隧道
- 调用外部编辑器编辑 tunnel JSON
- 复制隧道访问地址到剪贴板
- 创建 `Custom`、`Testing`、`Ephemeral` 三类隧道

## 运行依赖

- 构建依赖：Go `1.23+`
- 运行 `Custom` / `Testing` 隧道需要：`cloudflared`
- 后台运行隧道需要：`screen`

前置准备见 `docs/PREREQUISITES.md`。

## 安装

```bash
go build -o tunman ./cmd/tunman
sudo mv tunman /usr/local/bin/
```

## 使用

### TUI 模式

```bash
tunman
```

### CLI 模式

```bash
tunman version
```

说明：当前 CLI 框架已存在，但 `list`、`status`、`start`、`stop` 仍在完善中；日常管理建议优先使用 TUI。

## 常用快捷键

### 全局

| 按键 | 功能 |
|------|------|
| `q` / `Ctrl+C` | 退出 |
| `Esc` / `b` | 返回上一页 |
| `Ctrl+R` | 刷新状态 |
| `?` | 打开帮助页 |

### List 页面

| 按键 | 功能 |
|------|------|
| `j/k` 或 `↑/↓` | 上下选择 |
| `/` | 搜索 |
| `f` | 切换分类过滤 |
| `s/x/r` | 启动 / 停止 / 重启 |
| `Enter` / `l` | 打开日志页 |
| `a` | 打开创建页 |
| `e` | 外部编辑器编辑 |
| `d` | 打开删除页 |
| `y` | 复制 URL |
| `g/G` | 跳到首条 / 末条 |

### Create 页面

| 按键 | 功能 |
|------|------|
| `j/k` / `Tab` | 切换字段 |
| `Enter` / `Space` | 编辑字段或提交 |
| `←/→` | 切换隧道分类 |
| `Ctrl+S` | 提交创建 |

### Logs 页面

| 按键 | 功能 |
|------|------|
| `j/k` | 滚动日志 |
| `PgUp/PgDn` | 按页滚动 |
| `g/G` | 跳到首行 / 末行 |
| `t` | 切换跟随模式 |

## 数据目录

运行数据默认保存在 `~/.tunman/`：

```text
~/.tunman/
├── config.json
├── tunnels.json
├── logs/
└── configs/
```

## 文档

- `docs/PREREQUISITES.md`：Cloudflare / `cloudflared` / 域名准备
- `docs/USAGE.md`：详细操作说明
- `docs/PRD.md`：产品与交互设计说明

## 当前状态

- TUI 主流程已可用
- CLI 仍是补全中的能力
- 项目仍在迭代，文档已按当前实现同步
