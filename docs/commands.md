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
| `--version` | 显示 cmdguard 版本信息，同时显示底层命令版本 |
| `--help` | 显示 cmdguard 帮助（含保护级别说明），同时显示底层命令帮助 |

### 示例

```bash
# 验证 alias 是否生效
rm --check
# 输出: [cmdguard] 防护已生效 — rm 正在通过 cmdguard 运行

# 查看版本（同时显示 cmdguard 和底层 rm 版本）
rm --version
# cmdguard 0.4.0
# rm (GNU coreutils) 9.2
# ...

# 预览匹配结果
rm --dry-run -rf ~/Downloads/temp
# [cmdguard] ⚠️ 匹配规则: ~/Downloads/**
# [cmdguard] --dry-run 模式，未执行任何操作

# 查看详细执行信息
rm --verbose -rf ~/Downloads/temp
# [cmdguard] 匹配规则: ~/Downloads/** (级别: warn)
# [cmdguard] 备份: /Users/xxx/Downloads/temp
# [cmdguard] 执行: /bin/rm -rf /Users/xxx/Downloads/temp

# 查看帮助（同时显示 cmdguard 和底层 rm 帮助）
rm --help
```

---

## `cmdguard init [--force] [--dry-run]`

初始化环境。幂等安全，可重复执行。

| 选项 | 说明 |
|:----|:----|
| `--force`, `-f` | 强制覆盖已有配置文件和包装脚本（旧文件打包到 `~/.cmdguard/backup/init-<时间戳>.zip`） |
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
| `--id ID` | 按 ID 精确恢复 |
| `--interactive` | 交互式选择（默认） |
| `--dry-run` | 预览要恢复的文件，不实际恢复 |

```bash
cmdguard undo                    # 交互式选择
cmdguard undo --id abc123        # 按 ID 精确恢复
cmdguard undo --dry-run          # 预览恢复
cmdguard list --json | cmdguard undo   # 管道模式
```

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

---

## `cmdguard config`

查看当前配置。

```bash
cmdguard config
```

---

## `cmdguard help`

显示帮助信息。

---

## `cmdguard version`

显示版本信息。
