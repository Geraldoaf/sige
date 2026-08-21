package sandbox

import (
	"encoding/json"
	"testing"

	seccomp "github.com/seccomp/libseccomp-golang"
)

// TestBaselineSeccompProfileParses não instala o filtro (isso exige
// CAP_SYS_ADMIN ou NO_NEW_PRIVS, ver checkPrivileged em sandbox_test.go) —
// só garante que o JSON embutido continua válido e resolve pra uma
// allowlist não-vazia de syscalls reconhecidos pelo kernel/libseccomp atual.
// Sem isso, um baseline-seccomp.json corrompido/vazio só seria descoberto em
// runtime, como o daemon inteiro recusando subir.
func TestBaselineSeccompProfileParses(t *testing.T) {
	var profile seccompProfile
	if err := json.Unmarshal(baselineSeccompProfileJSON, &profile); err != nil {
		t.Fatalf("baseline-seccomp.json inválido: %v", err)
	}

	if len(profile.Syscalls) == 0 {
		t.Fatal("baseline-seccomp.json não tem nenhum grupo de syscalls")
	}

	resolved := 0
	for _, group := range profile.Syscalls {
		for _, name := range group.Names {
			if _, err := seccomp.GetSyscallFromName(name); err == nil {
				resolved++
			}
		}
	}

	if resolved == 0 {
		t.Fatal("nenhum syscall do baseline-seccomp.json foi resolvido pelo libseccomp instalado")
	}

	for _, required := range []string{"mount", "pivot_root", "unshare", "setuid", "setgid", "chroot", "prctl"} {
		found := false
		for _, group := range profile.Syscalls {
			for _, name := range group.Names {
				if name == required {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("baseline-seccomp.json não permite %q, necessário pro daemon/internal-launch", required)
		}
	}
}
