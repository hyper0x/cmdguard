# Changelog

## v0.3.0 (2026-06-06)

- 新增 `--check` 标志，用于验证 alias 是否生效
- 新增 `--dry-run` 标志（rm/mv/chmod/init/undo/vault clean）
- 新增 `--verbose` 标志（显示详细执行信息）
- 新增 `--version`/`--help` 支持（被托管的 rm/mv/chmod 命令，同时显示底层命令信息）
- 修复 `findRealCommand` 跳过包装脚本目录，防止无限递归
- 修复 `--dry-run`/`--verbose` 等 cmdguard 专属参数透传到底层命令
- Release 版本不显示 commit hash
- 文档拆分：`docs/commands.md`、`docs/configuration.md`、`docs/vault.md`
- 文档补全：`CMDGUARD_CONFIG_DIR`、`--check`、`--verbose`、`init --dry-run`、`undo --interactive`、`undo --dry-run`
- 新增 GitHub Actions：CI（go vet + go test）、Release（交叉编译 + 自动发布）

## v0.2.0 (2026-06-06)

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
