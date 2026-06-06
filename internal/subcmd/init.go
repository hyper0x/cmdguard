package subcmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
)

// backupFiles creates a zip archive of all files to be overwritten
func backupFiles(files []string) (string, error) {
	backupDir := filepath.Join(config.ConfigDir(), "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	ts := time.Now().Format("20060102_150405")
	zipPath := filepath.Join(backupDir, fmt.Sprintf("init-%s.zip", ts))

	z, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer z.Close()

	w := zip.NewWriter(z)
	defer w.Close()

	cfgDir := config.ConfigDir()

	for _, src := range files {
		in, err := os.Open(src)
		if err != nil {
			return "", err
		}
		defer in.Close()

		info, err := in.Stat()
		if err != nil {
			return "", err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return "", err
		}
		// Preserve relative path from ~/.cmdguard/
		rel, err := filepath.Rel(cfgDir, src)
		if err != nil {
			rel = filepath.Base(src)
		}
		header.Name = rel
		header.Method = zip.Deflate

		out, err := w.CreateHeader(header)
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(out, in); err != nil {
			return "", err
		}
	}

	return zipPath, nil
}

// RunInit handles the "init" command
func RunInit(args []string) {
	force := false
	for _, a := range args {
		if a == "--force" || a == "-f" {
			force = true
		}
	}

	cfgDir := config.ConfigDir()
	binDir := filepath.Join(cfgDir, "bin")
	logDir := filepath.Join(cfgDir, "log")
	vaultDir := filepath.Join(cfgDir, "vault")

	// 1. Create directory structure
	dirs := []string{cfgDir, binDir, logDir, vaultDir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[cmdguard] 错误: 创建目录 %s 失败: %v\n", d, err)
			os.Exit(1)
		}
		fmt.Printf("✓ 创建目录 %s\n", d)
	}

	// 2. Collect files to backup (only when --force and they exist)
	var toBackup []string
	cfgPath := config.ConfigPath()
	_, cfgStatErr := os.Stat(cfgPath)
	cfgExists := !os.IsNotExist(cfgStatErr)

	if force && cfgExists {
		toBackup = append(toBackup, cfgPath)
	}

	for _, cmd := range []string{"rm", "mv", "chmod"} {
		scriptPath := filepath.Join(binDir, cmd)
		_, err := os.Stat(scriptPath)
		if force && !os.IsNotExist(err) {
			toBackup = append(toBackup, scriptPath)
		}
	}

	// 3. Backup if needed
	if len(toBackup) > 0 {
		zipPath, err := backupFiles(toBackup)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[cmdguard] 错误: 备份文件失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ 备份到 %s\n", zipPath)
	}

	// 4. Generate default config
	needWriteConfig := !cfgExists
	if force && cfgExists {
		needWriteConfig = true
	}

	if needWriteConfig {
		defaultCfg := `# cmdguard 配置文件
# 路径保护规则，支持 glob 模式（** 匹配任意层级）
# 按保护级别分组（从严格到宽松）：
#
#   reject           🚫  直接拒绝，不执行
#   confirm_double   🔒  警告 + 双层确认（输入 yes）→ 备份 → 执行
#   confirm          ❓  警告 + 单层确认（按 y）→ 备份 → 执行
#   warn             ⚠️  警告 + 备份 → 执行
#
# 规则匹配顺序：reject → confirm_double → confirm → warn
# 匹配到第一条即停止，所以更严格的规则应放在前面。

[protect]
reject = [
  # macOS/Linux 系统目录
  "/bin/**",
  "/boot/**",
  "/dev/**",
  "/etc/**",
  "/lib/**",
  "/lib64/**",
  "/proc/**",
  "/sbin/**",
  "/sys/**",
  "/usr/**",
  "/var/**",
  "/opt/**",
  "/System/**",
  "/Library/**",
  "/Applications/**",
  # 密钥文件 — 不可再生
  "*.key",
  "*.pem",
  "*.crt",
  "*.p12",
  "*.pfx",
  "*.asc",
  # 家目录关键配置 — 不可再生
  "~/.ssh/**",
  "~/.gnupg/**",
  "~/.aws/**",
]

confirm_double = [
  # 家目录应用数据 — 可清理但需双重确认
  "~/.config/**",
  "~/.local/share/**",
]

confirm = [
  # 文档归档 - 确认
  "~/Documents/archive/**",
]

warn = [
  # 下载目录 - 警告+备份
  "~/Downloads/**",
]

# 按命令覆盖
[protect.command.rm]
reject = [
  "~/Documents/不许删",
]

[protect.command.mv]
reject = [
  "~/Projects/important",
]

[protect.command.chmod]
reject = [
  "~/Projects/important",
]

# Vault 备份设置
[vault]
retention_days = 30
auto_purge = true
`
		if err := os.WriteFile(cfgPath, []byte(defaultCfg), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[cmdguard] 错误: 写入配置文件 %s 失败: %v\n", cfgPath, err)
			os.Exit(1)
		}
		if force {
			fmt.Printf("✓ 覆盖配置文件 %s\n", cfgPath)
		} else {
			fmt.Printf("✓ 创建配置文件 %s\n", cfgPath)
		}
	} else {
		fmt.Printf("• 配置文件已存在，跳过: %s\n", cfgPath)
	}

	// 5. Create wrapper scripts in ~/.cmdguard/bin/
	selfPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 错误: 获取自身路径失败: %v\n", err)
		os.Exit(1)
	}

	for _, cmd := range []string{"rm", "mv", "chmod"} {
		scriptPath := filepath.Join(binDir, cmd)
		_, err := os.Stat(scriptPath)
		needWriteScript := os.IsNotExist(err)

		if force && !os.IsNotExist(err) {
			needWriteScript = true
		}

		if needWriteScript {
			script := fmt.Sprintf(`#!/bin/bash
exec %s %s "$@"
`, selfPath, cmd)
			if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "[cmdguard] 错误: 创建包装脚本 %s 失败: %v\n", scriptPath, err)
				os.Exit(1)
			}
			if force {
				fmt.Printf("✓ 覆盖包装脚本 %s\n", scriptPath)
			} else {
				fmt.Printf("✓ 创建包装脚本 %s\n", scriptPath)
			}
		} else {
			fmt.Printf("• 包装脚本已存在，跳过: %s\n", scriptPath)
		}
	}

	// 6. Print shell integration guide
	fmt.Println()
	fmt.Println("✅ cmdguard 初始化完成！")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  集成方式")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("方式一：alias（防护人类误操作）")
	fmt.Println("  在 ~/.zshrc、~/.bashrc 或 ~/.bash_profile 中添加：")
	fmt.Println()
	fmt.Println("    alias rm='cmdguard rm'")
	fmt.Println("    alias mv='cmdguard mv'")
	fmt.Println("    alias chmod='cmdguard chmod'")
	fmt.Println()
	fmt.Println("  适用场景：交互式终端操作")
	fmt.Println("  注意：脚本和程序直接调用 /bin/rm 等不会经过 alias")
	fmt.Println()
	fmt.Println("方式二：PATH 劫持（防护 AI 智能体误操作）")
	fmt.Println("  在 ~/.zshrc、~/.bashrc 或 ~/.bash_profile 中添加：")
	fmt.Println()
	fmt.Println("    export PATH=\"$HOME/.cmdguard/bin:$PATH\"")
	fmt.Println()
	fmt.Println("  适用场景：需要拦截脚本或 AI agent 中的 rm/mv/chmod 调用")
	fmt.Println("  注意：直接调用 /bin/rm 等绝对路径仍会绕过")
	fmt.Println()
	fmt.Println("两种方式可以同时使用，不会冲突。")
	fmt.Println("修改 shell 配置文件（~/.zshrc、~/.bashrc、~/.bash_profile 等）后执行 source 使其生效，或重启终端。")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
