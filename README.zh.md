<div align="center">

# cmdguard

**命令防护工具** — 为 `rm`、`mv`、`chmod` 提供安全防护、自动备份和操作撤销。

[![Go Report Card](https://goreportcard.com/badge/github.com/hyper0x/cmdguard)](https://goreportcard.com/report/github.com/hyper0x/cmdguard)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[English](README.md)

</div>

---

## 概述

cmdguard 包装危险命令（`rm`、`mv`、`chmod`），防止误操作导致数据丢失。

- 🚫 **四级防护** — reject、confirm_double、confirm、warn
- 💾 **自动备份** — 危险操作前自动备份到 vault
- ↩️ **操作撤销** — 通过 `cmdguard undo` 恢复被删除/覆盖的文件
- 📋 **审计日志** — 所有操作永久记录，支持搜索过滤
- 🤖 **智能体感知** — 对 AI 智能体和自动化场景有专门处理（见下）
- ⚙️ **TOML 配置** — 支持 glob 路径模式、命令级覆盖

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

将 cmdguard 的 wrapper 目录放到 PATH 最前面，智能体执行 `rm`/`mv`/`chmod` 时会先命中 cmdguard 包装脚本：

```bash
export PATH="$(cmdguard config --bin-dir):$PATH"
export CMDGUARD_NONINTERACTIVE=1   # 跳过 5/10 秒等待
```

`cmdguard config --bin-dir` 会以裸路径形式打印 wrapper 目录（默认 `~/.cmdguard/bin`，
也可能是你自定义的位置），即使将来调整目录布局，上面的 export 也仍然正确。

两种方式可以共存。

### cmdguard 防护不到的情况

无法拦截绕过 PATH 查找的调用，例如 `/bin/rm /etc/passwd`。cmdguard 的定位是"防护 + 审计 + 可恢复"，不是"绝对拦截"。

---

## AI 智能体使用方式

智能体调用受保护路径时，cmdguard 拒绝挂起等待交互。智能体需要：

1. **设置环境变量** 声明自己处于非交互模式：
   ```bash
   export CMDGUARD_NONINTERACTIVE=1
   ```
   这会跳过 5/10 秒的等待。**但不等于放行** —— 操作仍然会被拒绝。

2. **如果操作确实安全**，附加 `--bypass` 标识：
   ```bash
   rm /path/to/file --bypass=<host>/<platform>/<agent>/<task>
   ```
   标识必须恰好包含 4 段：

   | 段名     | 含义                  | 示例                   |
   |----------|-----------------------|------------------------|
   | host     | 主机名/机器别名       | `mac-studio`           |
   | platform | 智能体平台            | `qwenpaw`, `cursor`    |
   | agent    | 智能体 ID             | `ai_research`          |
   | task     | 任务简称              | `cleanup-tmp-dirs`     |

   允许字符：`[a-zA-Z0-9._-]`。空段、角括号、模板占位词（`host`、`agent`、`task`、`xxx`、`foo`、`todo` 等）会被拒绝。

每次 bypass 都记入审计日志，便于追溯到具体智能体/任务。

详见 [docs/commands.md](docs/commands.md#--bypass)。

---

## 快速开始

### 安装

```bash
# 方式一：下载预编译二进制（推荐）
# 从 Releases 页面下载对应平台的二进制

# 方式二：从源码编译
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard
make install   # 安装到 $GOBIN，自动注入版本号
```

### 初始化

```bash
cmdguard init
```

创建 `~/.cmdguard/{config.toml, bin/, log/, vault/}` 并打印集成指南。幂等，可重复执行。`--force` 覆盖旧文件（自动打包到 `~/.cmdguard/backup/`）。

---

## 命令速览

| 命令 | 说明 |
|:----|:----|
| `rm/mv/chmod <args...>` | 以防护模式运行命令 |
| `init [--force] [--dry-run]` | 初始化环境 |
| `list [选项]` | 列出审计日志 |
| `undo [选项]` | 从 vault 恢复操作 |
| `vault list [--json]` | 列出全部 vault 备份 |
| `vault clean [--dry-run]` | 清理过期 vault 备份 |
| `config [--default \| --raw \| --bin-dir]` | 多视角查看配置 |
| `path` | 展示 cmdguard 目录结构 |
| `help` / `version` | 帮助/版本 |

完整命令参考：[docs/commands.md](docs/commands.md)

---

## 配置文件

路径：`~/.cmdguard/config.toml`（可通过 `CMDGUARD_CONFIG_DIR` 自定义）。

```toml
[protect]
reject         = ["/etc/**", "/private/**", "~/.ssh/**"]
confirm_double = ["~/.config/**"]
confirm        = ["~/Documents/**", "~/Desktop/**"]
warn           = ["~/Downloads/**"]

[vault]
retention_days = 7
auto_purge     = true

[guard]
confirm_timeout        = 5    # 单确认等待秒数
confirm_double_timeout = 10   # 双确认每步等待秒数
```

> **配置文件对 cmdguard 是只读的**。cmdguard 运行时不会修改它；只有 `cmdguard init --force` 才会覆盖（自动备份为 zip）。

详细说明：[docs/configuration.md](docs/configuration.md)

---

## Vault 与撤销恢复

- 备份目录：`~/.cmdguard/vault/<时间戳>_<ID>/`
- 保留天数：默认 30 天（可配置）
- 恢复命令：`cmdguard undo [--id <id>] [--interactive] [--dry-run]`
- 日志：永久保留，JSON 格式，按天分文件

详细说明：[docs/vault.md](docs/vault.md)

---

## 从源码构建

```bash
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard
make build    # 构建到 ./dist，从 git tag 注入版本号
make install  # 安装到 $GOBIN
```

---

## 常见问题

**能否绕过 cmdguard？**
能。`/bin/rm /etc/passwd` 跳过 PATH 查找。cmdguard 是"防护 + 审计 + 可恢复"，不是"绝对拦截"。

**`confirm` 和 `confirm_double` 有什么区别？**
`confirm` 单次按 `y` 即可。`confirm_double` 先按 `y`，再完整输入 `yes`，专门防止疲劳误按。

**配置文件被覆盖怎么办？**
`cmdguard init` 不带 `--force` 不会覆盖。带 `--force` 时旧文件打包到 `~/.cmdguard/backup/init-<时间戳>.zip`。

**`CMDGUARD_NONINTERACTIVE` 智能体应该在哪里设？**
在智能体的 shell 启动脚本或环境配置里。设置后该环境下所有 cmdguard 调用都会跳过等待，直接走拒绝/bypass 分支。

**如何卸载？**
```bash
# 先从 shell 配置中移除 alias / PATH 设置，然后：
trash ~/.cmdguard         # 或没装 trash 用 rm -rf
rm $(which cmdguard)
```

---

## 许可证

[MIT](LICENSE)
