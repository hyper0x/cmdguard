# 命令参考

## `cmdguard rm/mv/chmod <args...>`

以防护模式运行命令。通过 alias 或 PATH 劫持自动调用，也可手动使用。

```bash
# 直接使用
cmdguard rm -rf ~/Downloads/temp

# 通过 alias
alias rm='cmdguard rm'
rm -rf ~/Downloads/temp
```

### 特殊选项

| 选项 | 说明 |
|:----|:----|
| `--check` | 验证 cmdguard 防护是否生效（不会执行真实命令） |
| `--dry-run` | 预览匹配结果，不执行（确认规则匹配是否符合预期后再实际执行） |
| `--verbose` | 显示详细执行信息（匹配规则、备份路径、实际命令等） |
| `--version` | 显示 cmdguard 版本信息，同时尝试显示底层命令版本（GNU 工具有效，BSD 静默跳过） |
| `--help` | 显示 cmdguard 帮助（含保护级别说明），同时尝试显示底层命令帮助 |
| `--bypass=<id>` | 强制放行 guarded 路径（见下方 [`--bypass`](#--bypass) 章节） |

### 示例

```bash
# 验证 alias 是否生效
rm --check
# 输出: [cmdguard] guard active - rm is running through cmdguard

# 查看版本（同时显示 cmdguard 和底层 rm 版本）
rm --version
# cmdguard 0.6.0
# rm (GNU coreutils) 9.2
# ...

# 预览匹配结果
rm --dry-run -rf ~/Downloads/temp

# 查看详细执行信息
rm --verbose -rf ~/Downloads/temp

# 查看帮助
rm --help
```

---

## `--bypass`

**用途：** 调用方（AI 智能体或人类）主动声明"这次操作我已审视过，请放行"。

**格式：** `--bypass=<platform>/<agent>/<task>` - 必须 3 段路径形式：

| 段名     | 含义                      | 示例                   |
|----------|---------------------------|------------------------|
| platform | 智能体平台                | `qwenpaw`, `cursor`, `claude-code` |
| agent    | 智能体 ID 或 `manual`     | `ai_research`, `coding` |
| task     | 任务简称                  | `cleanup-tmp-dirs`     |

**校验规则：**

- 恰好 3 段，`/` 分隔
- 每段匹配 `[a-zA-Z0-9._-]+`，不允许空段
- 总长度 ≥ 10 字符
- 段不能是模板占位词（`platform`、`agent`、`task`、`xxx`、`foo`、`todo` 等）
- 不允许角括号 `<>` 或花括号 `{}`（防止直接复制模板）

**示例：**

```bash
# ✅ 正确
rm /tmp/cache --bypass=qwenpaw/ai_research/cleanup-cache
mv old.txt /Users/x/Documents/new.txt --bypass=cursor/default/refactor
rm ~/Downloads/old.zip --bypass=manual/haolin/cleanup-downloads

# ❌ 被拒（直接复制模板）
rm /tmp/cache --bypass='<platform>/<agent>/<task>'
rm /tmp/cache --bypass=platform/agent/task

# ❌ 被拒（占位词/格式不合规）
rm /tmp/cache --bypass=xxx/yyy/zzz
rm /tmp/cache --bypass=abc/def        # 段不足 3
```

**工作方式：** 当命令目标命中 **guarded** 路径且未提供 `--bypass` 时，cmdguard
立即拒绝操作并打印期望的 `--bypass` 格式引导。提供有效 `--bypass` 后，cmdguard
将目标文件备份到 vault，记录操作日志（含完整 bypass 标识），然后执行。

**备份失败始终致命。** 如果 vault 备份失败，操作中止（exit 1），无论调用方是
智能体还是人类。undo 安全网已断，没有回滚能力就不应继续执行。

**审计：** 每次 bypass 都记入日志，`cmdguard list` 显示 `[bypass:qwenpaw/ai_research/cleanup-cache]` 标签，可追溯到具体智能体/任务。

---

## `cmdguard init [--force] [--dry-run]`

初始化环境。幂等安全，可重复执行。

| 选项 | 说明 |
|:----|:----|
| `--force` | 强制覆盖已有配置文件和包装脚本（旧文件打包到 `~/.cmdguard/backup/init-<时间戳>.zip`） |
| `--dry-run` | 预览操作，不实际执行 |

```bash
cmdguard init                    # 首次初始化
cmdguard init --force            # 强制覆盖
cmdguard init --dry-run          # 预览
cmdguard init --force --dry-run  # 预览强制覆盖
```

---

## `cmdguard list [选项]`

列出操作日志。

| 选项 | 说明 |
|:----|:----|
| `--recent N` | 最近 N 条（默认 20） |
| `--since D` | 从多久前开始（如 `"2h"`、`"7d"`） |
| `--cmd C` | 按命令过滤（`rm`/`mv`/`chmod`） |
| `--path P` | 按路径关键词过滤 |
| `--json` | JSON 格式输出（支持管道） |

```bash
cmdguard list                    # 最近 20 条
cmdguard list --recent 50        # 最近 50 条
cmdguard list --since 7d         # 最近 7 天
cmdguard list --cmd rm           # 只显示 rm 操作
cmdguard list --path Documents   # 按路径关键词过滤
cmdguard list --json             # JSON 格式输出
```

---

## `cmdguard undo [选项]`

恢复操作。

| 选项 | 说明 |
|:----|:----|
| `--id ID` | 按 ID 精确恢复（支持短 ID 前缀匹配） |
| `--interactive` | 交互式选择 |
| `--dry-run` | 预览要恢复的文件，不实际恢复 |

```bash
cmdguard undo --id abc123              # 按 ID 精确恢复
cmdguard undo --interactive            # 交互式选择
cmdguard undo --id abc123 --dry-run    # 预览恢复
cmdguard list --json | cmdguard undo   # 管道模式
```

---

## `cmdguard vault list [--json]`

列出 vault 中所有备份。

| 选项 | 说明 |
|:----|:----|
| `--json` | 输出 JSON 数组（便于脚本处理） |

```bash
cmdguard vault list
cmdguard vault list --json
```

默认文本输出包含 ID、命令、创建时间、过期时间和原始目标路径。按时间倒序。

---

## `cmdguard vault clean [--dry-run]`

清理过期 vault 备份。

| 选项 | 说明 |
|:----|:----|
| `--dry-run` | 预览要清理的备份，不实际删除 |

```bash
cmdguard vault clean              # 清理过期备份
cmdguard vault clean --dry-run    # 预览
```

直接执行 `cmdguard vault`（不带子命令）会打印用法并以 exit 1 退出 --
`clean` 属于破坏性操作，不应被静默作为默认行为。

---

## `cmdguard config [--default | --raw | --bin-dir]`

查看不同视角的配置。这三个选项互斥。

| 选项 | 说明 |
|:----|:----|
| _(无)_ | 合并后的有效配置（默认值 + 用户覆盖） |
| `--default` | 仅展示内置默认配置 |
| `--raw` | 展示磁盘上 `config.toml` 的原始内容 |
| `--bin-dir` | 仅打印 wrapper 脚本目录的路径 |

```bash
cmdguard config              # 实际生效的配置
cmdguard config --default    # 假设没有 config.toml 时的样子
cmdguard config --raw        # 磁盘上配置文件的原文
cmdguard config --bin-dir    # /home/alice/.cmdguard/bin
```

`--bin-dir` 主要用于 shell 拼接：

```bash
export PATH="$(cmdguard config --bin-dir):$PATH"
```

---

## `cmdguard path`

展示 cmdguard 的目录结构 -- 配置文件、日志目录、vault 目录、bin 目录，
并附带文件数量和大小。

```bash
cmdguard path
```

日志部分最多展示最新 5 个文件，避免长期运行后输出过长，并在末尾给出总文件数。

---

## `cmdguard help` / `cmdguard version`

显示帮助 / 版本信息。
