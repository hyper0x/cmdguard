// Package errs centralises stderr printing helpers used by the
// subcommand layer. Before this package existed, every subcommand
// inlined the pattern:
//
//	fmt.Fprintf(os.Stderr, msg.FmtErr(template)+"\n", args...)
//
// which carried a subtle bug: msg.FmtErr does its own Sprintf, so
// any %v / %s / %q in `template` was consumed there with no args,
// producing %!v(MISSING) noise that the outer Fprintf then mangled
// further. The helpers below take (template, args) once and format
// in a single pass, eliminating the double-format hazard.
package subcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hyper0x/cmdguard/internal/msg"
)

// errExit prints a [cmdguard] error message to stderr and exits 1.
// It is the canonical "fatal user-visible error" path for subcommands.
//
// The template is the bare error text, e.g. "failed to load config: %v".
// Do NOT embed the [cmdguard] tag or "error:" prefix in the template —
// they are added by msg.FmtErr.
func errExit(format string, args ...any) {
	fmt.Fprintln(os.Stderr, msg.FmtErr(format, args...))
	os.Exit(1)
}

// errPrint prints a [cmdguard] error message to stderr WITHOUT
// exiting. Use this when the caller wants to continue running
// (e.g. printing extra guidance after the error before exiting
// at a different point).
func errPrint(format string, args ...any) {
	fmt.Fprintln(os.Stderr, msg.FmtErr(format, args...))
}

// warn prints a [cmdguard] warning to stderr. Used for non-fatal
// degraded conditions (backup failed, log load failed, etc.).
func warn(format string, args ...any) {
	fmt.Fprintln(os.Stderr, msg.FmtWarn(format, args...))
}

// rejectUnknownArg is the shared "this subcommand doesn't accept
// that token" path. It picks the right error message based on
// whether the token looks like a flag (starts with "-") or a
// positional argument, so users see "unknown flag --recnet" vs.
// "unexpected argument foo" — the diagnostic actually matches what
// they typed. Always exits 1 via errExit.
func rejectUnknownArg(arg, subcommand string) {
	if strings.HasPrefix(arg, "-") {
		errExit(msg.ErrUnknownFlag, arg, subcommand)
	}
	errExit(msg.ErrUnexpectedArg, arg, subcommand)
}
