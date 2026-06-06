# Vault 备份机制

cmdguard 有自己的 vault 备份系统（不依赖系统回收站）。

## 工作原理

1. 当 `confirm`、`confirm_double` 或 `warn` 级别的操作被执行时，目标文件先备份到 vault
2. 备份目录：`~/.cmdguard/vault/<操作ID>/`
3. 备份包含原始文件的完整副本和元数据
4. 原始命令执行完毕后，vault 中的备份用于 `undo` 恢复

## 为什么不用系统回收站？

- macOS 回收站无法处理 `mv` 覆盖场景
- 多次删除同名文件时回收站会混淆
- 回收站没有与操作日志关联的元数据

## 自动清理

- `retention_days`：备份保留天数（默认 30）
- `auto_purge`：每次执行防护命令时自动清理过期备份（默认开启）
- 日志永久保留（体积小，有审计价值）

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
