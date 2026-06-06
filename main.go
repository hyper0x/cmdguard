package main

import (
	"fmt"
	"os"

	subcmd "github.com/hyper0x/cmdguard/internal/subcmd"
)

var version = "dev"   // set via -ldflags at build time
var commit  = "none"  // set via -ldflags at build time

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	sub := os.Args[1]

	switch sub {
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		printVersion()
	case "init":
		subcmd.RunInit(os.Args[2:])
	case "list":
		subcmd.RunList(os.Args[2:])
	case "undo":
		subcmd.RunUndo(os.Args[2:])
	case "vault":
		subcmd.RunVault(os.Args[2:])
	case "config":
		subcmd.RunConfig(os.Args[2:])
	default:
		// sub is a command name like rm, mv, chmod
		guardCmd := sub
		args := os.Args[2:]
		subcmd.Version = version
		subcmd.Commit = commit
		subcmd.RunGuard(guardCmd, args)
	}
}

func printVersion() {
	fmt.Printf("cmdguard %s (commit: %s)\n", version, commit)
}

func printUsage() {
	fmt.Print(`cmdguard — 命令防护工具

用法:
  cmdguard <command> [选项]

命令:
  rm <args...>       以 rm 模式运行（别名: alias rm='cmdguard rm'）
                      附加 --check 可验证 alias 是否生效
                      --dry-run 预览匹配结果，不执行
                      --version 显示版本（含底层命令版本）
                      --help 显示帮助（含底层命令帮助）
  mv <args...>       以 mv 模式运行
  chmod <args...>    以 chmod 模式运行
  init [--force]     初始化环境（创建目录、配置文件、包装脚本）
                      --force 覆盖已有文件，旧文件打包到 ~/.cmdguard/backup/
                      --dry-run 预览操作，不实际执行
  list [选项]         列出操作日志
  undo [选项]         恢复操作
                      --dry-run 预览要恢复的文件，不实际恢复
  vault clean        清理过期 vault 备份
  config             查看当前配置
  help, -h, --help   显示帮助信息
  version, -v, --version  显示版本信息

list 选项:
  --recent N    最近 N 条（默认 20）
  --since D     从多久前开始（如 "2h"、"7d"）
  --cmd C       按命令过滤（rm/mv/chmod）
  --path P      按路径关键词过滤
  --json        输出 JSON 格式

undo 选项:
  --id ID       按 ID 精确恢复
  --interactive 交互式选择（默认）

vault 选项:
  --dry-run     只列出要删的，不实际删除

保护级别:
  reject           🚫  直接拒绝，不执行
  confirm_double   🔒  警告 + 双层确认（输入 yes）→ 备份 → 执行
  confirm          ❓  警告 + 单层确认（按 y）→ 备份 → 执行
  warn             ⚠️  警告 + 备份 → 执行

配置文件: ~/.cmdguard/config.toml
日志目录: ~/.cmdguard/log/
Vault目录: ~/.cmdguard/vault/

环境变量:
  CMDGUARD_CONFIG_DIR  自定义配置目录（默认 ~/.cmdguard）
`)
}
