# 配置文件

路径：`~/.cmdguard/config.toml`

> **环境变量：** `CMDGUARD_CONFIG_DIR` 可自定义配置目录（含 `config.toml`、`log/`、`vault/`、`bin/`）。设置后所有路径均以此目录为根。
>
> **只读契约：** 配置文件对 cmdguard 程序本身是**只读**的。cmdguard 运行时不会修改、重写或自动清理它。只有 `cmdguard init --force` 才会覆盖（同时把旧文件打包到 `~/.cmdguard/backup/init-<时间戳>.zip`）。如果删错路径不存在了，cmdguard 不会"帮你"剔除条目——这是为了保证你的安全策略只反映你写下的内容。

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
retention_days = 7
auto_purge = true

[guard]
confirm_timeout = 5          # 单确认等待秒数
confirm_double_timeout = 10  # 双确认每步等待秒数
```

> ⚠️ **配置优先级说明：** 当配置文件定义了 `[protect]` 段时，文件的规则**直接生效**——内置默认规则不会叠加。如果你不想要任何 reject 规则，写 `reject = []`。
>
> 如果文件**没有** `[protect]` 段，则使用内置默认保护规则（这样只写了 `[vault]` 的配置文件仍然有默认保护）。
>
> `[vault]` 和 `[guard]` 使用**字段级合并**：只有你显式写的字段覆盖默认值，没写的字段保留内置默认值。

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

## `[guard]` 段——交互超时

```toml
[guard]
confirm_timeout = 5          # 默认 5 秒
confirm_double_timeout = 10  # 默认 10 秒（每步）
```

| 键 | 默认值 | 说明 |
|:---|:-----:|:----|
| `confirm_timeout` | `5` | `confirm` 提示等待用户按键的秒数 |
| `confirm_double_timeout` | `10` | `confirm_double` **每一步**等待的秒数（按 `y` 一步 + 输入 `yes` 一步） |

**特殊值：**
- `0` = 禁用超时，永远等待（直到用户按键或 Ctrl+C 中止）。仅建议在不会有任何自动化命中 confirm 路径的个人本机使用，否则任何漏掉 env 设置的智能体调用会让进程永久挂起。

**超时触发后的行为：** 自动降级到非交互拒绝路径，输出与 `CMDGUARD_NONINTERACTIVE=1` / stdin 非 TTY 相同的 `--bypass` 引导信息，并在日志中记录 `timeout waiting for confirmation (Xs)`。

**为什么需要超时？**
`isTerminal()` 在某些智能体沙箱中会误判 stdin 为真 TTY（因为分配了伪终端）。如果没有超时，进程会永远挂在 `Are you sure?` 提示上。超时机制是为了在这些"假人"场景下兜底降级。

## 环境变量

| 环境变量 | 用途 |
|:--------|:----|
| `CMDGUARD_CONFIG_DIR` | 自定义配置目录（含 `config.toml`、`log/`、`vault/`、`bin/`），默认 `~/.cmdguard` |
| `CMDGUARD_NONINTERACTIVE` | 任意非空值即生效：跳过 `confirm`/`confirm_double` 的交互等待，直接走拒绝+bypass 引导路径。**不等于放行**——仍需 `--bypass=<id>` 才能真正执行 |

`CMDGUARD_NONINTERACTIVE` 是给 AI 智能体和 CI 用的。设置后即时拒绝（0 延迟），不再等 5/10 秒。详见 [docs/commands.md](commands.md#--bypass)。
