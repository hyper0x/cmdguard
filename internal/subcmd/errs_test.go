package subcmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hyper0x/cmdguard/internal/msg"
)

// TestErrPrint_NoDoubleFormat is the regression test for the
// pre-existing bug that motivated this whole errs.go file:
//
//	fmt.Fprintf(os.Stderr, msg.FmtErr(template)+"\n", args...)
//
// produced output like:
//
//	[cmdguard] error: invalid --since value %!!(string=7days)q(MISSING) ...
//
// because msg.FmtErr did its own Sprintf with no args, eagerly
// consuming the %q in the template, and the outer Fprintf then
// formatted a string containing %!q(MISSING) literally.
//
// The errPrint helper does Sprintf in a single pass with the args,
// so %q renders correctly. We test that contract directly: an error
// message printed through errPrint must not contain "%!" (Go's
// missing-argument noise marker).
func TestErrPrint_NoDoubleFormat(t *testing.T) {
	stderr := captureStderr(t, func() {
		errPrint("invalid --since value %q (use formats like 30m, 2h, 7d)", "7days")
	})
	if strings.Contains(stderr, "%!") {
		t.Fatalf("errPrint produced double-format noise: %q", stderr)
	}
	if !strings.Contains(stderr, `"7days"`) {
		t.Fatalf("errPrint did not interpolate %%q argument: %q", stderr)
	}
	if !strings.HasPrefix(stderr, msg.TagCmdguard+" error:") {
		t.Errorf("errPrint missing [cmdguard] error: prefix: %q", stderr)
	}
}

// TestWarn_NoDoubleFormat is the same regression check for warnings.
// We feed an error value through %v plus a string through %s — the
// two-argument case is the one that previously printed worst.
func TestWarn_NoDoubleFormat(t *testing.T) {
	stderr := captureStderr(t, func() {
		warn("backup of %s failed: %v", "/etc/foo", errors.New("disk full"))
	})
	if strings.Contains(stderr, "%!") {
		t.Fatalf("warn produced double-format noise: %q", stderr)
	}
	if !strings.Contains(stderr, "/etc/foo") || !strings.Contains(stderr, "disk full") {
		t.Fatalf("warn did not interpolate args: %q", stderr)
	}
	if !strings.HasPrefix(stderr, msg.TagCmdguard+" warning:") {
		t.Errorf("warn missing [cmdguard] warning: prefix: %q", stderr)
	}
}

// TestErrPrint_NoArgs covers the no-arg shape: a template with no
// format verbs (e.g. errPrint("something went wrong")) must print
// the bare text without any noise. Previously this also worked,
// because there were no %v/%q to consume — but we lock the contract
// in case someone refactors errPrint to require args.
func TestErrPrint_NoArgs(t *testing.T) {
	stderr := captureStderr(t, func() {
		errPrint("something went wrong")
	})
	if strings.Contains(stderr, "%!") {
		t.Fatalf("errPrint with no args still produced noise: %q", stderr)
	}
	if !strings.Contains(stderr, "something went wrong") {
		t.Errorf("errPrint did not emit literal text: %q", stderr)
	}
}

// captureStderr runs fn with os.Stderr redirected to a pipe and
// returns the captured output. We use a real pipe (not a
// bytes.Buffer assigned to os.Stderr) because the helpers under
// test write through fmt.Fprintln(os.Stderr, ...) — they take the
// CURRENT value of os.Stderr at call time, so we must swap the
// global before calling.
//
// errExit is intentionally not exercised here because it calls
// os.Exit(1) and would tear down the test process; covering its
// formatting would require a subprocess test, which is heavier
// than the value adds (errExit is a one-line wrapper around
// errPrint).
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w

	// Drain the pipe in a goroutine — fmt.Fprintln writes are tiny
	// and unbuffered, but we're paranoid about pipe buffer fills
	// in case someone extends the test with bigger payloads.
	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, r)
	}()

	fn()

	_ = w.Close()
	os.Stderr = orig
	wg.Wait()
	_ = r.Close()
	return buf.String()
}
