# Changelog

## v0.2.0 (2026-06-06)

### 新增
- `confirm_double` 保护级别（双层确认，需输入 `yes`）
- `cmdguard init` 子命令（创建目录、生成配置、创建包装脚本、打印集成指南）
- `init --force` 备份旧文件到 `~/.cmdguard/backup/init-<时间戳>.zip`
- 单元测试：`internal/config/`、`internal/guard/`

### 修复
- `execOriginal` 挂起问题（去掉 `c.Stdin = os.Stdin` 绑定）
- `list --json` ID 截断（JSON 输出完整 ID，表格输出截断 8 字符）
- `FindByID` 前缀匹配（表格截断 ID 也能定位）
- vault `FindBackupDir`/`BackupExists` 前缀匹配
- 管道解析：支持表格和 JSON 两种格式
- 环境变量名 `CMGGUARD_CONFIG_DIR` → `CMDGUARD_CONFIG_DIR`
- `init` 输出：已有目录/文件显示 `• 已存在` 而非 `✓ 创建`
- `source` 指令通用化（不写死 `~/.zshrc`）

### 文档
- README：替换「灵感来源」为「设计思路」，涵盖人类和 AI 智能体防护
- README：去重集成说明，FAQ 精简
- `init` 输出：集成指南同步更新

## v0.1.0 (2026-06-06)

### 新增
- 首次发布
- 四级防护：reject、confirm_double、confirm、warn
- 命令包装：rm、mv、chmod
- 自动备份到 vault
- 撤销恢复（undo）
- 审计日志（JSON 格式，按天分文件）
- vault 自动清理（可配置保留天数）
- TOML 配置文件，glob 路径模式
- 支持按命令覆盖保护规则
