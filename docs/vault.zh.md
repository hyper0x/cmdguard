# Vault 备份机制

cmdguard 有自己的 vault 备份系统（不依赖系统回收站）。

## 工作原理

1. 当 `guarded` 级别的操作附带有效 `--bypass` 执行时，目标文件先备份到 vault
2. 备份目录：`~/.cmdguard/vault/<时间戳>_<操作ID>/`
3. 备份包含原始文件的完整副本和元数据
4. 原始命令执行完毕后，vault 中的备份用于 `undo` 恢复
5. 被 `reject` 的操作只记日志，不写入 vault（操作未执行，无需备份）

## 为什么不用系统回收站？

- macOS 回收站无法处理 `mv` 覆盖场景
- 多次删除同名文件时回收站会混淆
- 回收站没有与操作日志关联的元数据
- 跨平台一致：Linux/macOS/Windows 都用同样的 vault 机制

## 自动清理

- `retention_days`：备份保留天数（默认 7）
- `auto_purge`：每次执行防护命令时自动清理过期备份（默认开启）
- 日志永久保留（体积小，有审计价值）

也可以手动清理：

```bash
cmdguard vault clean              # 清理过期备份
cmdguard vault clean --dry-run    # 预览
```

## 撤销恢复

`undo` 命令从 vault 中恢复文件：

```bash
# 按 ID 精确恢复（支持短 ID 前缀匹配）
cmdguard list                          # 找到目标 ID（短 ID 即可）
cmdguard undo --id abc12345

# 交互式选择
cmdguard undo --interactive

# 预览
cmdguard undo --id abc12345 --dry-run

# 管道模式（list 输出直接进 undo）
cmdguard list --json | cmdguard undo
```

### 恢复行为

| 原始操作 | 恢复行为 |
|:--------|:--------|
| `rm file` | 将文件从 vault 复制回原位置 |
| `mv src dst` | 将目标文件从 vault 恢复（注意：源文件不会自动移回） |
| `chmod file` | 恢复文件的原始权限 |

## 审计日志

所有经过 cmdguard 的操作（包括被拒绝的、`allow` 的、`bypass` 的、`undo` 本身、`vault-clean`）都会记录到 `~/.cmdguard/log/`。

- 格式：JSON，按天分文件（`YYYY-MM-DD.jsonl`）
- 保留：**永久**（不随 vault 清理而删除）
- 字段：操作 ID、时间、命令、动作、目标路径、匹配规则、消息、bypass 标识

```json
{
  "id": "abc123456789",
  "timestamp": "2026-06-06T12:00:00+08:00",
  "command": "rm",
  "action": "guarded",
  "targets": "/Users/x/Documents/old.txt",
  "rule": "/Users/x/Documents/**",
  "message": "path matches protection rule '/Users/x/Documents/**'",
  "bypass": "qwenpaw/ai_research/cleanup-cache"
}
```

`bypass` 字段使审计可追溯到具体的智能体和任务。

### 旧版日志条目

v0.14 之前版本创建的日志条目可能包含 `confirm`、`confirm_double` 或 `warn`
作为 `action` 值。这些条目仍可被 `cmdguard list` 和 `cmdguard undo` 正常读取。
旧版常量保留用于向后兼容。
