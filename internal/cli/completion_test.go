package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionCmdRegistered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "completion" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("completion command not registered on RootCmd")
	}
}

func TestCompletionValidArgs(t *testing.T) {
	expected := []string{"bash", "zsh", "fish", "powershell"}
	for _, shell := range expected {
		found := false
		for _, arg := range completionCmd.ValidArgs {
			if arg == shell {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("completion ValidArgs missing %q", shell)
		}
	}
}

func TestCompletionBash(t *testing.T) {
	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	defer RootCmd.SetOut(nil)

	err := completionCmd.RunE(completionCmd, []string{"bash"})
	if err != nil {
		t.Fatalf("completion bash error = %v", err)
	}
	got := buf.String()
	if len(got) == 0 {
		t.Fatal("completion bash produced no output")
	}
	if !strings.Contains(got, "bash") || (!strings.Contains(got, "complete") && !strings.Contains(got, "compgen")) {
		t.Fatalf("completion bash output does not look like bash completion; got %d bytes", len(got))
	}
}

func TestCompletionZsh(t *testing.T) {
	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	defer RootCmd.SetOut(nil)

	err := completionCmd.RunE(completionCmd, []string{"zsh"})
	if err != nil {
		t.Fatalf("completion zsh error = %v", err)
	}
	got := buf.String()
	if len(got) == 0 {
		t.Fatal("completion zsh produced no output")
	}
}

func TestCompletionFish(t *testing.T) {
	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	defer RootCmd.SetOut(nil)

	err := completionCmd.RunE(completionCmd, []string{"fish"})
	if err != nil {
		t.Fatalf("completion fish error = %v", err)
	}
	got := buf.String()
	if len(got) == 0 {
		t.Fatal("completion fish produced no output")
	}
}

func TestCompletionPowershell(t *testing.T) {
	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	defer RootCmd.SetOut(nil)

	err := completionCmd.RunE(completionCmd, []string{"powershell"})
	if err != nil {
		t.Fatalf("completion powershell error = %v", err)
	}
	got := buf.String()
	if len(got) == 0 {
		t.Fatal("completion powershell produced no output")
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	defer RootCmd.SetOut(nil)

	err := completionCmd.RunE(completionCmd, []string{"tcsh"})
	if err == nil {
		t.Fatal("completion tcsh should return an error")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("error = %q, want 'unsupported shell'", err.Error())
	}
}
