# TunMan 使用教程

本文档只保留“如何使用 TunMan”。

如果你还没准备运行环境，请先阅读 `docs/PREREQUISITES.md`。

## 1. 构建与启动

在项目根目录执行：

```bash
go build -o tunman ./cmd/tunman
./tunman
```

如果希望放到系统路径：

```bash
sudo mv tunman /usr/local/bin/
tunman
```

## 2. 运行模式

### 2.1 TUI 模式

直接运行：

```bash
tunman
```

这是当前推荐的主要使用方式。

### 2.2 CLI 模式

当前 CLI 还在完善中，已明确可用的是：

```bash
tunman version
```

`list`、`start`、`stop`、`status` 入口已经存在，但目前仍以 TUI 使用为主。

## 3. 页面说明

TunMan 当前包含这些页面：

- `List`：查看隧道列表并执行主要操作
- `Create`：创建新隧道
- `Delete`：确认删除隧道
- `Logs`：查看日志输出
- `Help`：查看快捷键帮助

## 4. 全局快捷键

- `Ctrl+R`：刷新状态
- `?`：打开帮助页
- `Esc` / `b`：返回上一页
- `q`：退出程序

说明：退出时会自动清理临时隧道。

## 5. List 页面常用操作

- `j/k` 或 `↑/↓`：上下选择
- `/`：进入搜索模式
- `f`：切换分类过滤（All / Custom / Testing / Ephemeral）
- `s`：启动选中隧道
- `x`：停止选中隧道
- `r`：重启选中隧道
- `Enter` 或 `l`：打开日志页
- `a`：进入创建页
- `e`：使用外部编辑器编辑隧道 JSON
- `d`：进入删除页
- `y`：复制当前隧道 URL
- `g/G`：跳到首条 / 末条

## 6. 创建隧道

### 6.1 Create 页面快捷键

- `j/k` 或 `Tab`：切换字段
- `Enter` 或 `Space`：编辑字段、切换分类或提交创建
- `←/→`：切换隧道分类
- `Ctrl+S`：提交创建

### 6.2 三种隧道的填写方式

#### `Custom`

需要填写：

- `Base Domain`
- `Subdomain`
- `Port`
- `Description`（可选）

适合长期固定域名，例如：

- 根域名：`example.com`
- 子域名前缀：`api`
- 最终访问地址：`https://api.example.com`

#### `Testing`

需要填写：

- `Base Domain`
- `Port`
- `Description`（可选）

创建时 TunMan 会自动生成一个测试用前缀。

#### `Ephemeral`

需要填写：

- `Port`
- `Description`（可选）

这种模式适合临时联调，不需要你准备域名或 Cloudflare 登录。

### 6.3 创建后的行为

- `Custom` / `Testing`：创建完成后会尝试自动启动
- `Ephemeral`：创建后可在列表中直接查看和管理

创建完成后，建议回到 `List` 页面确认状态，并进入 `Logs` 页面检查输出。

## 7. 查看日志

在 `Logs` 页面中：

- `j/k`：滚动日志
- `PgUp/PgDn`：按页滚动
- `g/G`：跳到首行 / 末行
- `t`：切换跟随模式

如果日志为空，先确认隧道是否已经启动。

## 8. 删除隧道

建议流程：

1. 在 `List` 页面选中目标隧道
2. 按 `d` 进入删除页
3. 按 `y` 确认删除
4. 按 `n`、`Esc` 或 `b` 取消删除

## 9. 数据文件位置

TunMan 运行数据位于：

```text
~/.tunman/
├── config.json
├── tunnels.json
├── logs/
└── configs/
```

## 10. 常见问题

### 10.1 启动失败

优先检查：

- `screen` 是否已安装并在 `PATH` 中
- `Custom` / `Testing` 模式下，`cloudflared` 是否已安装
- `Custom` / `Testing` 模式下，是否已执行 `cloudflared tunnel login`

### 10.2 固定域名隧道不可用

优先检查：

- 根域名是否已接入 Cloudflare
- Cloudflare 授权是否有效
- 本地服务端口是否真的在运行

### 10.3 剪贴板复制失败

这通常是系统剪贴板工具不可用，不影响隧道本身运行。
