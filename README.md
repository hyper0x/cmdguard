<div align="center">

# cmdguard

**命令防护工具** — 为 `rm`、`mv`、`chmod` 提供安全防护、自动备份和操作撤销。

[![Go Report Card](https://goreportcard.com/badge/github.com/hyper0x/cmdguard)](https://goreportcard.com/report/github.com/hyper0x/cmdguard)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

## 概述

cmdguard 包装危险命令（`rm`、`mv`、`chmod`），防止误操作导致数据丢失。

- 🚫 **三级防护** — reject（拒绝）、confirm/confirm_double（确认）、warn（警告）
- 💾 **自动备份** — 执行危险操作前自动备份到 vault
- ↩️ **撤销恢复** — 通过 `undo` 恢复被删除/覆盖的文件
- 📋 **审计日志** — 所有操作永久记录，支持搜索和过滤
- ⚙️ **灵活配置** — TOML 格式，glob 路径模式

---

## 设计思路

### 防护人类误操作

在 `~/.zshrc` 或 `~/.bashrc` 中添加 alias：

```bash
alias rm='cmdguard rm'
alias mv='cmdguard mv'
alias chmod='cmdguard chmod'
```

### 防护 AI 智能体误操作

将 `~/.cmdguard/bin/` 放到 PATH 最前面：

```bash
export PATH="$HOME/.cmdguard/bin:$PATH"
```

### 防护不到的情况

无法拦截直接使用绝对路径的调用（如 `/bin/rm`）。cmdguard 的定位是"防护 + 审计 + 可恢复"，不是"绝对拦截"。

两种方式可以同时使用，不会冲突。

---

## 快速开始

### 安装

```bash
# 方式一：下载预编译二进制（推荐）
# 从 Releases 页面下载对应平台的二进制

# 方式二：从源码编译
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard
go build -o cmdguard .
sudo mv cmdguard /usr/local/bin/
```

### 初始化

```bash
cmdguard init
```

一次完成：创建目录结构 → 生成默认配置 → 创建包装脚本 → 打印集成指南。

> 更多选项：`cmdguard init --force`（覆盖）、`cmdguard init --dry-run`（预览）
> 详见 [命令参考](docs/commands.md#cmdguard-init---force---dry-run)

---

## 命令速览

| 命令 | 说明 |
|:----|:----|
| `rm/mv/chmod <args...>` | 以防护模式运行命令 |
| `init [--force] [--dry-run]` | 初始化环境 |
| `list [选项]` | 列出操作日志 |
| `undo [选项]` | 恢复操作 |
| `vault clean [--dry-run]` | 清理过期 vault 备份 |
| `config` | 查看当前配置 |
| `help` | 显示帮助 |
| `version` | 显示版本 |

> 完整命令参考：[docs/commands.md](docs/commands.md)

---

## 配置文件

路径：`~/.cmdguard/config.toml`（可通过 `CMDGUARD_CONFIG_DIR` 环境变量自定义）

```toml
[protect]
reject = ["/etc/**", "~/.ssh/**"]
confirm_double = ["~/.config/**"]
confirm = ["~/Documents/archive/**"]
warn = ["~/Downloads/**"]

[vault]
retention_days = 30
auto_purge = true
```

> 详细配置说明：[docs/configuration.md](docs/configuration.md)

---

## Vault 与撤销恢复

- 备份目录：`~/.cmdguard/vault/<操作ID>/`
- 保留天数：默认 30 天（可配置）
- 恢复命令：`cmdguard undo`
- 日志：永久保留，JSON 格式，按天分文件

> 详细说明：[docs/vault.md](docs/vault.md)

---

## 从源码构建

```bash
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard
go build -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo 'dev') -X main.commit=$(git rev-parse --short HEAD)" -o cmdguard .
```

---

## 常见问题

### Q: 绕过 cmdguard 怎么办？

cmdguard 是"防护 + 审计 + 可恢复"，不是"绝对拦截"。刻意绕过（如 `/bin/rm`）无法阻止。

### Q: `confirm_double` 和 `confirm` 有什么区别？

`confirm_double` 需要两次确认，第二次必须完整输入 `yes`，防止疲劳误按。

### Q: 配置文件被覆盖了怎么办？

`cmdguard init` 不会覆盖已有配置（除非 `--force`）。`--force` 时旧文件打包到 `~/.cmdguard/backup/init-<时间戳>.zip`。

### Q: 如何完全卸载？

```bash
# 从 shell 配置中移除 alias/PATH 设置
rm -rf ~/.cmdguard
rm /usr/local/bin/cmdguard
```

---

## 许可证

[MIT](LICENSE)
