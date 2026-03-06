# TunMan v2.0 产品需求文档

> Go + Bubble Tea 重构版本，纯键盘操作，多页面架构。

---

## 1. 产品定位

TunMan 是一个用于管理 Cloudflare Tunnel 的终端工具（TUI），提供比官方 `cloudflared` 更便捷的隧道生命周期管理。

**核心原则：**
- 纯键盘操作，无鼠标依赖
- 单二进制分发，运行依赖保持最少
- 简洁界面，类似 `htop`/`ranger` 的交互体验

---

## 2. 隧道分类

| 分类 | 标识 | 用途 | 特点 |
|------|------|------|------|
| **Custom** | `custom` | 固定子域名服务 | 手工指定根域名与前缀 |
| **Testing** | `testing` | 临时测试域名 | 自动生成短前缀 |
| **Ephemeral** | `ephemeral` | 本地快速联调 | 随机域名，退出即清理 |

---

## 3. 功能规格

### 3.1 隧道管理（CRUD）

| 功能 | 操作 | 说明 |
|------|------|------|
| Create | 表单交互 | 按分类填写 `Base Domain`、`Subdomain`、`Port` 等字段 |
| Read | 列表展示 | 分类过滤、搜索、状态实时刷新 |
| Update | 外部编辑器 | 唤起 `$EDITOR` 编辑 JSON |
| Delete | 独立确认页 | 进入删除页后再确认删除 |

### 3.2 生命周期控制

| 操作 | 快捷键 | 说明 |
|------|--------|------|
| Start | `s` | 启动隧道（screen 后台） |
| Stop | `x` | 停止隧道 |
| Restart | `r` | 重启隧道 |
| Tail | `t` | 实时跟踪日志 |

### 3.3 临时隧道

| 能力 | 当前入口 | 说明 |
|------|----------|------|
| Quick Temp | TUI Create 页面选择 `Ephemeral` | 创建随机域名隧道 |
| Auto-cleanup | 退出时自动清理 | 工具退出时停止所有临时隧道 |

---

## 4. 界面架构

### 4.1 页面设计

| 页面 | 当前入口 | 功能 |
|------|----------|------|
| **List** | 默认页 | 隧道列表，主操作页面 |
| **Create** | `a` | 创建新隧道表单 |
| **Delete** | `d` | 删除确认页面 |
| **Logs** | `Enter` / `l` | 查看日志 |
| **Help** | `?` | 快捷键帮助 |

### 4.2 全局快捷键

| 按键 | 功能 |
|------|------|
| `q` / `Ctrl+C` | 退出 |
| `Esc` / `b` | 返回 |
| `Ctrl+R` | 刷新状态 |
| `?` | 打开帮助 |

### 4.3 List 页面操作

| 按键 | 功能 |
|------|------|
| `j/k` / `↑/↓` | 选择隧道 |
| `/` | 搜索过滤 |
| `f` | 切换分类 |
| `s` | 启动 |
| `x` | 停止 |
| `r` | 重启 |
| `Enter` / `l` | 查看日志 |
| `a` | 打开创建页 |
| `e` | 编辑配置 |
| `d` | 打开删除页 |
| `y` | 复制 URL |

---

## 5. 数据存储

### 5.1 目录结构

```
~/.tunman/
├── config.json      # 工具配置
├── tunnels.json     # 隧道配置
└── logs/            # 日志文件
    ├── {id}.log
    └── ephemeral-*.log
```

### 5.2 配置格式

**config.json**
```json
{
  "version": "2.0",
  "auto_refresh": true,
  "refresh_interval": 5,
  "editor": "vim"
}
```

**tunnels.json**
```json
{
  "tunnels": [
    {
      "id": "newapi",
      "category": "protected",
      "name": "NewAPI",
      "port": 3000,
      "subdomain": "dev.example.com",
      "tunnel_id": "xxx-xxx",
      "created_at": "2025-02-10T10:00:00Z"
    }
  ]
}
```

---

## 6. CLI 接口

```bash
# TUI 模式（默认）
tunman
tunman list

# CLI 模式
tunman start <id>
tunman stop <id>
tunman restart <id>
tunman status
tunman logs <id> [--follow]
tunman temp <port> [--expire <minutes>]
tunman create
tunman delete <id>
tunman edit <id>
```

---

## 7. 技术栈

- **语言**: Go 1.23+
- **TUI 框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **组件库**: [Bubbles](https://github.com/charmbracelet/bubbles) (list, textinput, viewport)
- **样式**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **JSON**: `encoding/json` (Go 标准库)
- **进程管理**: GNU Screen

---

## 8. 前置依赖

使用 TunMan 前需要手动完成：

1. **域名** - 购买并接入 Cloudflare
2. **Cloudflare 账号** - 免费版即可
3. **cloudflared** - 官方客户端安装
4. **Tunnel 创建** - 通过 cloudflared 创建并配置 DNS

详见：[PREREQUISITES.md](./PREREQUISITES.md)

---

*版本: v2.0*  
*状态: 实现中*
