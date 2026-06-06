# 配置文件

路径：`~/.cmdguard/config.toml`

> **环境变量：** `CMDGUARD_CONFIG_DIR` 可自定义配置目录（含 `config.toml`、`log/`、`vault/`、`bin/`）。设置后，所有路径均以此目录为根。

## 基本结构

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
  "~/Documents/**",
  "~/Desktop/**",
]
warn = [
  "~/Downloads/**",
]

# 按命令覆盖
[protect.command.rm]
reject = [
  "~/.config/**",
]

# mv 保护目标路径（最后一个参数），对 ~/Downloads/** 设为 reject 防止误移入覆盖文件
[protect.command.mv]
reject = [
  "~/Downloads/**",
]

# chmod 修改权限，对应用配置目录设为 reject（全局规则中 ~/.config/** 是 confirm_double）
[protect.command.chmod]
reject = [
  "~/.config/**",
]

[vault]
retention_days = 30
auto_purge = true
```

## 路径模式

支持 glob 模式：

| 模式 | 含义 | 示例 |
|:----|:----|:----|
| `**` | 匹配任意层级 | `/etc/**` 匹配 `/etc/` 下所有文件和子目录 |
| `*` | 匹配文件名中的任意字符 | `*.key` 匹配所有 `.key` 文件 |
| `?` | 匹配单个字符 | `file.???` 匹配 `file.txt` 但不匹配 `file.html` |
| `~` | 自动展开为用户家目录 | `~/.ssh/**` → `/Users/xxx/.ssh/**` |

## 默认配置

首次运行或 `cmdguard init` 生成的默认配置包含：

- **reject：** 系统目录（`/bin/`、`/etc/`、`/usr/`、`/private/` 等）、密钥文件（`*.key`、`*.pem` 等）、家目录关键配置（`~/.ssh/`、`~/.gnupg/`、`~/.aws/`）
- **confirm_double：** 家目录应用数据（`~/.config/`、`~/.local/share/`）
- **confirm：** 文档目录（`~/Documents/`）、桌面文件（`~/Desktop/`）
- **warn：** 下载目录（`~/Downloads/`）

## 保护级别

| 级别 | 图标 | 行为 | 适用场景 |
|:----|:---:|:----|:--------|
| `reject` | 🚫 | 直接拒绝，不执行 | 系统目录、密钥文件、不可再生的配置 |
| `confirm_double` | 🔒 | 警告 + 双层确认（输入 `yes`）→ 备份 → 执行 | 应用数据目录，可清理但需谨慎 |
| `confirm` | ❓ | 警告 + 单层确认（按 `y`）→ 备份 → 执行 | 文档归档等偶尔需要清理的路径 |
| `warn` | ⚠️ | 警告 + 备份 → 执行 | 下载目录等临时文件，仅提醒 |

**规则匹配顺序：** `reject → confirm_double → confirm → warn`，匹配到第一条即停止。
