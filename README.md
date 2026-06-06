<div align="center">

# cmdguard

**命令防护工具** — 为 `rm`、`mv`、`chmod` 提供安全防护、自动备份和撤销恢复

[![Go Report Card](https://goreportcard.com/badge/github.com/hyper0x/cmdguard)](https://goreportcard.com/report/github.com/hyper0x/cmdguard)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

## 概述

cmdguard 是一个 CLI 工具，包装危险命令（`rm`、`mv`、`chmod`），防止误操作导致数据丢失。

**核心能力：**

- 🚫 **三级防护** — reject（拒绝）、confirm/confirm_double（确认）、warn（警告）
- 💾 **自动备份** — 执行危险操作前自动备份到 vault
- ↩️ **撤销恢复** — 通过 `undo` 命令恢复被删除/覆盖的文件
- 📋 **审计日志** — 所有操作永久记录，支持搜索和过滤
- ⚙️ **灵活配置** — TOML 格式，glob 路径模式，支持按命令覆盖

---

## 设计思路

cmdguard 的设计源于一个核心问题：**如何防止危险命令（rm/mv/chmod）误操作导致数据丢失？**

这里的"误操作"有两个来源：

### 防护人类误操作

人类在终端中操作时，最容易犯错：
- `rm -rf /etc /tmp/x`（多打了个空格）
- `rm -rf ~/Projects /tmp/x`（本意是删临时目录，结果家目录也没了）
- 疲劳时手快按了 `y`

**防护方式：** 在 `~/.zshrc` 或 `~/.bashrc` 中添加 alias，让交互式终端的 `rm`/`mv`/`chmod` 自动走 cmdguard：

```bash
alias rm='cmdguard rm'
alias mv='cmdguard mv'
alias chmod='cmdguard chmod'
```

### 防护 AI 智能体误操作

AI agent 在执行任务时，可能调用 `rm`、`mv`、`chmod` 等命令。由于 agent 没有人类的判断力，误操作风险更高。

**防护方式：** 将 `~/.cmdguard/bin/` 放到 PATH 的最前面，让 agent 调用的 `rm`/`mv`/`chmod`（包括脚本中的调用）自动走 cmdguard：

```bash
export PATH="$HOME/.cmdguard/bin:$PATH"
```

### 防护不到的情况

无论 alias 还是 PATH 劫持，都无法拦截直接使用绝对路径的调用（如 `/bin/rm`）。这是操作系统层面的限制，cmdguard 的定位是"防护 + 审计 + 可恢复"，不是"绝对拦截"。

两种方式可以同时使用，不会冲突。修改 shell 配置文件（`~/.zshrc`、`~/.bashrc`、`~/.bash_profile` 等）后执行 `source` 使其生效，或重启终端。

---

## 快速开始

### 安装

```bash
# 方式一：下载预编译二进制（推荐）
# 从 Releases 页面下载对应平台的二进制，放到 PATH 中

# 方式二：从源码编译
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard
go build -o cmdguard .
sudo mv cmdguard /usr/local/bin/
```

### 初始化

```bash
cmdguard init                    # 首次初始化
cmdguard init --force            # 强制覆盖已有配置（旧文件打包到 ~/.cmdguard/backup/）
```

一次完成：
1. 创建 `~/.cmdguard/` 目录结构（`bin/`、`log/`、`vault/`）
2. 生成默认配置文件 `~/.cmdguard/config.toml`
3. 创建包装脚本 `~/.cmdguard/bin/{rm,mv,chmod}`
4. 打印 shell 集成指南

> **幂等安全**：已存在的配置文件和包装脚本不会被覆盖（除非使用 `--force`）。使用 `--force` 时，旧文件自动打包到 `~/.cmdguard/backup/init-<时间戳>.zip`。

### 集成到 Shell

`cmdguard init` 完成后会打印完整的集成指南。详细说明请参考[设计思路](#设计思路)中的防护方式。

---

## 保护级别

cmdguard 提供四级防护，按严格程度降序排列：

| 级别 | 图标 | 行为 | 适用场景 |
|:----|:---:|:----|:--------|
| `reject` | 🚫 | 直接拒绝，不执行 | 系统目录、密钥文件、不可再生的配置 |
| `confirm_double` | 🔒 | 警告 + 双层确认（输入 `yes`）→ 备份 → 执行 | 应用数据目录，可清理但需谨慎 |
| `confirm` | ❓ | 警告 + 单层确认（按 `y`）→ 备份 → 执行 | 文档归档等偶尔需要清理的路径 |
| `warn` | ⚠️ | 警告 + 备份 → 执行 | 下载目录等临时文件，仅提醒 |

**规则匹配顺序：** `reject → confirm_double → confirm → warn`，匹配到第一条即停止。

---

## 配置文件

路径：`~/.cmdguard/config.toml`

> **环境变量：** `CMDGUARD_CONFIG_DIR` 可自定义配置目录（含 `config.toml`、`log/`、`vault/`、`bin/`）。设置后，所有路径均以此目录为根。例如：
> ```bash
> export CMDGUARD_CONFIG_DIR=/data/cmdguard
> # 配置文件 → /data/cmdguard/config.toml
> # 日志目录 → /data/cmdguard/log/
> # vault目录 → /data/cmdguard/vault/
> # 包装脚本 → /data/cmdguard/bin/
> ```

### 基本结构

```toml
[protect]
reject = [
  "/etc/**",
  "~/.ssh/**",
]
confirm_double = [
  "~/.config/**",
]
confirm = [
  "~/Documents/archive/**",
]
warn = [
  "~/Downloads/**",
]

# 按命令覆盖
[protect.command.rm]
reject = [
  "~/Documents/不许删",
]

[vault]
retention_days = 30
auto_purge = true
```

### 路径模式

支持 glob 模式：

| 模式 | 含义 | 示例 |
|:----|:----|:----|
| `**` | 匹配任意层级 | `/etc/**` 匹配 `/etc/` 下所有文件和子目录 |
| `*` | 匹配文件名中的任意字符 | `*.key` 匹配所有 `.key` 文件 |
| `?` | 匹配单个字符 | `file.???` 匹配 `file.txt` 但不匹配 `file.html` |
| `~` | 自动展开为用户家目录 | `~/.ssh/**` → `/Users/xxx/.ssh/**` |

### 默认配置

首次运行或 `cmdguard init` 生成的默认配置包含：

- **reject：** 系统目录（`/bin/`、`/etc/`、`/usr/` 等）、密钥文件（`*.key`、`*.pem` 等）、家目录关键配置（`~/.ssh/`、`~/.gnupg/`、`~/.aws/`）
- **confirm_double：** 家目录应用数据（`~/.config/`、`~/.local/share/`）
- **confirm：** 文档归档（`~/Documents/archive/`）
- **warn：** 下载目录（`~/Downloads/`）

---

## 命令参考

### `cmdguard rm/mv/chmod <args...>`

以防护模式运行命令。通过 alias 或 PATH 劫持自动调用，也可手动使用：

```bash
# 直接使用
cmdguard rm -rf ~/Downloads/temp

# 通过 alias
alias rm='cmdguard rm'
rm -rf ~/Downloads/temp
```

**特殊选项：**

| 选项 | 说明 |
|:----|:----|
| `--check` | 验证 cmdguard 防护是否生效（不会执行真实命令） |
| `--dry-run` | 预览匹配结果，不执行（确认规则匹配是否符合预期后再实际执行） |
| `--version` | 显示 cmdguard 版本信息，同时显示底层命令版本 |
| `--help` | 显示 cmdguard 帮助（含保护级别说明），同时显示底层命令帮助 |

示例：

```bash
# 验证 alias 是否生效
rm --check
# 输出: [cmdguard] 防护已生效 — rm 正在通过 cmdguard 运行

# 查看版本（同时显示 cmdguard 和底层 rm 版本）
rm --version
# cmdguard 0.3.0 (commit: abc1234)
# rm (GNU coreutils) 9.2
# ...

# 预览匹配结果
rm --dry-run -rf ~/Downloads/temp
# [cmdguard] ⚠️ 匹配规则: ~/Downloads/**
# [cmdguard] --dry-run 模式，未执行任何操作

# 查看帮助（同时显示 cmdguard 和底层 rm 帮助）
rm --help
```

### `cmdguard init [--force]`

初始化环境。幂等安全，可重复执行。

```bash
cmdguard init                    # 首次初始化
cmdguard init --force            # 强制覆盖已有配置（旧文件打包到 ~/.cmdguard/backup/）
```

### `cmdguard list [选项]`

列出操作日志。

```bash
cmdguard list                    # 最近 20 条
cmdguard list --recent 50        # 最近 50 条
cmdguard list --since 7d         # 最近 7 天
cmdguard list --cmd rm           # 只显示 rm 操作
cmdguard list --path Documents   # 按路径关键词过滤
cmdguard list --json             # JSON 格式输出（支持管道）
```

### `cmdguard undo [选项]`

恢复操作。

```bash
cmdguard undo                    # 交互式选择
cmdguard undo --id abc123        # 按 ID 精确恢复
cmdguard list --json | cmdguard undo   # 管道模式
```

### `cmdguard vault clean`

清理过期 vault 备份。

```bash
cmdguard vault clean              # 清理过期备份
cmdguard vault clean --dry-run    # 预览要清理的备份，不实际删除
```

### `cmdguard config`

查看当前配置。

```bash
cmdguard config
```

### `cmdguard help`

显示帮助信息。

### `cmdguard version`

显示版本信息。

---

## Vault 备份机制

cmdguard 有自己的 vault 备份系统（不依赖系统回收站）。

### 工作原理

1. 当 `confirm`、`confirm_double` 或 `warn` 级别的操作被执行时，目标文件先备份到 vault
2. 备份目录：`~/.cmdguard/vault/<操作ID>/`
3. 备份包含原始文件的完整副本和元数据
4. 原始命令执行完毕后，vault 中的备份用于 `undo` 恢复

### 为什么不用系统回收站？

- macOS 回收站无法处理 `mv` 覆盖场景
- 多次删除同名文件时回收站会混淆
- 回收站没有与操作日志关联的元数据

### 自动清理

- `retention_days`：备份保留天数（默认 30）
- `auto_purge`：每次执行防护命令时自动清理过期备份（默认开启）
- 日志永久保留（体积小，有审计价值）

---

## 撤销恢复

`undo` 命令从 vault 中恢复文件：

```bash
# 交互式选择
cmdguard undo

# 按 ID 精确恢复
cmdguard list --json
# 复制目标 ID，然后：
cmdguard undo --id abc123456789

# 管道模式
cmdguard list --json | cmdguard undo
```

### 恢复行为

| 原始操作 | 恢复行为 |
|:--------|:--------|
| `rm file` | 将文件从 vault 复制回原位置 |
| `mv src dst` | 将目标文件从 vault 恢复（注意：源文件不会自动移回） |
| `chmod file` | 恢复文件的原始权限 |

---

## 审计日志

所有操作（包括被拒绝的操作）都会记录到 `~/.cmdguard/log/`。

- 格式：JSON，按天分文件
- 保留：永久
- 内容：操作命令、时间、目标路径、匹配规则、操作结果

```json
{
  "id": "abc123456789",
  "timestamp": "2026-06-06T12:00:00+08:00",
  "command": "rm",
  "action": "reject",
  "targets": "/etc/passwd",
  "rule": "/etc/**",
  "message": "路径匹配保护规则 '/etc/**'"
}
```

---

## 从源码构建

```bash
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard

# 构建
go build -o cmdguard .

# 构建并注入版本信息
go build -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo 'dev') -X main.commit=$(git rev-parse --short HEAD)" -o cmdguard .

# 安装到 GOBIN
go install -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo 'dev') -X main.commit=$(git rev-parse --short HEAD)" .
```

---

## 常见问题

### Q: 绕过 cmdguard 怎么办？

cmdguard 的设计哲学是"防护 + 审计 + 可恢复"，不是"绝对拦截"。如果用户或程序刻意绕过（如直接调用 `/bin/rm`），cmdguard 无法阻止。但：
- 日志会记录所有走了 cmdguard 的操作
- vault 会备份走了 cmdguard 的操作

### Q: `confirm_double` 和 `confirm` 有什么区别？

`confirm_double` 需要两次确认，第二次必须完整输入 `yes`（而不是按 `y`），防止疲劳操作时手快误按。

### Q: 配置文件被覆盖了怎么办？

`cmdguard init` 不会覆盖已有配置文件（除非使用 `--force`）。如果不小心使用了 `--force`，备份文件保存在 `~/.cmdguard/backup/init-<时间戳>.zip`，解压即可恢复。

### Q: 如何完全卸载？

```bash
# 1. 从 shell 配置中移除 alias 或 PATH 设置
# 2. 删除 cmdguard 目录
rm -rf ~/.cmdguard
# 3. 删除二进制
rm /usr/local/bin/cmdguard
```

---

## 许可证

[MIT](LICENSE)
