# 配置文件

路径：`~/.cmdguard/config.toml`

> **环境变量：** `CMDGUARD_CONFIG_DIR` 可自定义配置目录（含 `config.toml`、`log/`、`vault/`、`bin/`）。设置后所有路径均以此目录为根。
>
> **只读契约：** 配置文件对 cmdguard 程序本身是**只读**的。cmdguard 运行时不会修改、重写或自动清理它。只有 `cmdguard init --force` 才会覆盖（同时把旧文件打包到 `~/.cmdguard/backup/init-<时间戳>.zip`）。如果删错路径不存在了，cmdguard 不会"帮你"剔除条目--这是为了保证你的安全策略只反映你写下的内容。

## 基本结构

```toml
[protect]
reject = [
  "/etc/**",
  "~/.ssh/**",
]
guarded = [
  "~/.config/**",
  "~/Documents/**",
  "~/Desktop/**",
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

# chmod 修改权限，对应用配置目录设为 reject（全局规则中 ~/.config/** 是 guarded）
[protect.command.chmod]
reject = [
  "~/.config/**",
]

[vault]
retention_days = 7
auto_purge = true
```

> ⚠️ **配置优先级说明：** 当配置文件定义了 `[protect]` 段时，文件的规则**直接生效**--内置默认规则不会叠加。如果你不想要任何 reject 规则，写 `reject = []`。
>
> 如果文件**没有** `[protect]` 段，则使用内置默认保护规则（这样只写了 `[vault]` 的配置文件仍然有默认保护）。
>
> `[vault]` 使用**字段级合并**：只有你显式写的字段覆盖默认值，没写的字段保留内置默认值。

## 路径模式

支持 glob 模式：

| 模式 | 含义 | 示例 |
|:----|:----|:----|
| `**` | 匹配任意层级 | `/etc/**` 匹配 `/etc/` 下所有文件和子目录 |
| `*` | 匹配文件名中的任意字符 | `*.key` 匹配所有 `.key` 文件 |
| `?` | 匹配单个字符 | `file.???` 匹配 `file.txt` 但不匹配 `file.html` |
| `~` | 自动展开为用户家目录 | `~/.ssh/**` -> `/Users/xxx/.ssh/**` |

## 默认配置

首次运行或 `cmdguard init` 生成的默认配置包含：

- **reject：** 系统目录（`/bin/`、`/etc/`、`/usr/`、`/private/` 等）、密钥文件（`*.key`、`*.pem` 等）、家目录关键配置（`~/.ssh/`、`~/.gnupg/`）
- **guarded：** 家目录应用数据（`~/.config/`、`~/.local/share/`）、文档目录（`~/Documents/`）、桌面文件（`~/Desktop/`）
- **allow：** 其他所有路径（默认--不匹配任何规则的路径直接放行）

## 保护级别

| 级别 | 图标 | 行为 | 适用场景 |
|:----|:---:|:----|:--------|
| `reject` | 🚫 | 始终拒绝，即使带 `--bypass` 也不执行。记日志。 | 系统目录、密钥文件、不可再生的配置 |
| `guarded` | 🔒 | 不带 `--bypass`：拒绝 + 记日志。带有效 `--bypass`：备份 -> 记日志 -> 执行。 | 应用数据目录、文档等可清理但需谨慎的路径 |
| `allow` | ✅ | 直接执行。记日志。 | 不匹配任何规则的路径 |

**规则匹配顺序：** `reject -> guarded`，匹配到第一条即停止。不匹配任何规则的路径默认放行。

## 环境变量

| 环境变量 | 用途 |
|:--------|:----|
| `CMDGUARD_CONFIG_DIR` | 自定义配置目录（含 `config.toml`、`log/`、`vault/`、`bin/`），默认 `~/.cmdguard` |

cmdguard 不再使用任何控制行为的环境变量。旧的 `CMDGUARD_AGENT_MODE` 和 `CMDGUARD_NONINTERACTIVE` 会被静默忽略--wrapper 脚本不再设置它们，代码也不再读取它们。如果你已有的 wrapper 导出了这些变量，仍可正常工作（变量只是被忽略）。运行 `cmdguard init --force` 可重新生成干净的 wrapper。

## 从 v0.14 之前版本迁移

如果你有旧的 `config.toml` 包含 `[guard]` 段或旧的保护级别，改动对照如下：

| 旧 | 新 | 说明 |
|:----|:----|:------|
| `confirm_double` | `guarded` | 在 `[protect]` 中改键名 |
| `confirm` | `guarded` | 合并到同一个 `guarded` 列表 |
| `warn` | （移除） | 删除这些路径，或如果需要备份则移到 `guarded` |
| `[guard]` 段 | （移除） | 整个段被忽略；不再有交互设置 |
| `CMDGUARD_AGENT_MODE` | （移除） | 不再需要；非交互是唯一模式 |
| `CMDGUARD_NONINTERACTIVE` | （移除） | 同上 |

TOML 解码器会静默忽略未知键，所以旧配置文件不会报错--但旧键不再有任何效果。`cmdguard init --force` 会写入使用新结构的干净配置。
