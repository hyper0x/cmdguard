# Changelog

## v0.2.0 (2026-06-06)

- 新增 `confirm_double` 保护级别（双层确认）
- 新增 `cmdguard init` 子命令（初始化环境，`--force` 备份旧文件）
- 新增 `cmdguard list --json` JSON 输出格式
- 修复管道模式（支持表格和 JSON 两种格式）
- 修复 `init` 输出：已有目录/文件显示 `• 已存在`
- 环境变量名 `CMGGUARD_CONFIG_DIR` → `CMDGUARD_CONFIG_DIR`
- 完善文档：设计思路、人类与 AI 智能体防护

## v0.1.0 (2026-06-06)

- 首次发布
- 四级防护：reject、confirm_double、confirm、warn
- 命令包装：rm、mv、chmod
- 自动备份到 vault，支持撤销恢复
- 审计日志（JSON 格式，按天分文件）
- TOML 配置文件，glob 路径模式
