package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yourusername/gogit/internal/repository"
)

func TestCheckoutCanSwitchBackAfterSkippingSubtree(t *testing.T) {
	root := enterCheckoutRepo(t)
	checkoutOK(t, os.Mkdir("docs", 0755))
	writeCheckoutFile(t, "docs/readme", "nested")
	writeCheckoutFile(t, "tracked", "base")
	checkoutOK(t, runAdd(nil, []string{"docs", "tracked"}))
	commitMessage = "base"
	checkoutOK(t, runCommit(nil, nil))

	refs := repository.NewRefs(root)
	base, err := refs.ResolveHead()
	checkoutOK(t, err)
	checkoutOK(t, refs.CreateBranch("target", base))
	checkoutOK(t, runCheckout(nil, []string{"target"}))
	checkoutOK(t, runCheckout(nil, []string{"main"}))
	if got := readCheckoutFile(t, "docs/readme"); got != "nested" {
		t.Fatalf("nested file = %q, want nested", got)
	}
}

func TestCheckoutCanSwitchBackAfterExecutableModeChange(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix executable bits")
	}
	root := enterCheckoutRepo(t)
	writeCheckoutFile(t, "script", "script")
	checkoutOK(t, runAdd(nil, []string{"script"}))
	commitMessage = "base"
	checkoutOK(t, runCommit(nil, nil))

	refs := repository.NewRefs(root)
	base, err := refs.ResolveHead()
	checkoutOK(t, err)
	checkoutOK(t, refs.CreateBranch("target", base))
	checkoutOK(t, refs.SetHead("target", true))
	checkoutOK(t, os.Chmod("script", 0755))
	checkoutOK(t, runAdd(nil, []string{"script"}))
	commitMessage = "make executable"
	checkoutOK(t, runCommit(nil, nil))

	checkoutOK(t, runCheckout(nil, []string{"main"}))
	info, err := os.Stat("script")
	checkoutOK(t, err)
	if info.Mode()&0111 != 0 {
		t.Fatal("script is executable on main")
	}
	checkoutOK(t, runCheckout(nil, []string{"target"}))
	info, err = os.Stat("script")
	checkoutOK(t, err)
	if info.Mode()&0111 == 0 {
		t.Fatal("script is not executable on target")
	}
}

func TestCheckoutProtectsLocalChanges(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		stage     bool
		detached  bool
		wantError string
	}{
		{name: "unstaged edit on same commit", path: "tracked", wantError: "local changes"},
		{name: "staged edit with detached target", path: "tracked", stage: true, detached: true, wantError: "staged changes"},
		{name: "staged addition outside target", path: "staged-only", stage: true, wantError: "staged changes"},
		{name: "untracked collision", path: "z-target-only", wantError: "untracked path"},
		{name: "clean checkout with unrelated untracked file", path: "unrelated", detached: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := enterCheckoutRepo(t)
			writeCheckoutFile(t, "tracked", "base")
			checkoutOK(t, runAdd(nil, []string{"tracked"}))
			commitMessage = "base"
			checkoutOK(t, runCommit(nil, nil))

			refs := repository.NewRefs(root)
			base, err := refs.ResolveHead()
			checkoutOK(t, err)
			target := base
			if tt.path != "tracked" || tt.detached {
				checkoutOK(t, refs.CreateBranch("target", base))
				checkoutOK(t, refs.SetHead("target", true))
				writeCheckoutFile(t, "tracked", "target")
				writeCheckoutFile(t, "z-target-only", "new file")
				checkoutOK(t, runAdd(nil, []string{"tracked", "z-target-only"}))
				commitMessage = "target"
				checkoutOK(t, runCommit(nil, nil))
				target, err = refs.ResolveHead()
				checkoutOK(t, err)
				checkoutOK(t, runCheckout(nil, []string{"main"}))
				// Checkout leaves files absent from the target in the working tree.
				checkoutOK(t, os.Remove("z-target-only"))
			} else {
				checkoutOK(t, refs.CreateBranch("target", base))
			}

			writeCheckoutFile(t, tt.path, "local edit")
			if tt.stage {
				checkoutOK(t, runAdd(nil, []string{tt.path}))
			}
			headPath := filepath.Join(".gogit", "HEAD")
			indexPath := filepath.Join(".gogit", "index")
			headBefore := readCheckoutFile(t, headPath)
			indexBefore := readCheckoutFile(t, indexPath)
			trackedBefore := readCheckoutFile(t, "tracked")

			arg := "target"
			if tt.detached {
				arg = target
			}
			err = runCheckout(nil, []string{arg})
			if tt.wantError == "" {
				checkoutOK(t, err)
				if got := readCheckoutFile(t, "tracked"); got != "target" {
					t.Fatalf("tracked file = %q, want target", got)
				}
				if got := readCheckoutFile(t, headPath); got != target+"\n" {
					t.Fatalf("HEAD = %q, want detached at %s", got, target)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("checkout error = %v, want %q", err, tt.wantError)
				}
				for path, want := range map[string]string{
					headPath: headBefore, indexPath: indexBefore, "tracked": trackedBefore,
				} {
					if got := readCheckoutFile(t, path); got != want {
						t.Errorf("checkout changed %s", path)
					}
				}
			}
			if got := readCheckoutFile(t, tt.path); got != "local edit" {
				t.Errorf("local edit in %s changed to %q", tt.path, got)
			}
		})
	}
}

func enterCheckoutRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cwd, err := os.Getwd()
	checkoutOK(t, err)
	checkoutOK(t, os.Chdir(root))
	previousMessage := commitMessage
	t.Cleanup(func() {
		commitMessage = previousMessage
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	checkoutOK(t, runInit(nil, nil))
	return root
}

func checkoutOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func writeCheckoutFile(t *testing.T, path, content string) {
	t.Helper()
	checkoutOK(t, os.WriteFile(path, []byte(content), 0644))
}

func readCheckoutFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	checkoutOK(t, err)
	return string(content)
}
