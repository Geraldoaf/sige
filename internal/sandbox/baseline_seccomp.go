package sandbox

import (
	_ "embed"
	"encoding/json"
	"fmt"

	seccomp "github.com/seccomp/libseccomp-golang"
)

//go:embed baseline-seccomp.json
var baselineSeccompProfileJSON []byte

type seccompRule struct {
	Names    []string `json:"names"`
	Action   string   `json:"action"`
	ErrnoRet int      `json:"errnoRet"`
}

type seccompProfile struct {
	DefaultErrnoRet int           `json:"defaultErrnoRet"`
	Syscalls        []seccompRule `json:"syscalls"`
}

// baselineSyscallNames retorna a lista de syscalls permitidas no perfil base embutido.
func baselineSyscallNames() ([]string, error) {
	var profile seccompProfile
	if err := json.Unmarshal(baselineSeccompProfileJSON, &profile); err != nil {
		return nil, fmt.Errorf("error parsing embedded baseline seccomp profile: %w", err)
	}

	var names []string
	for _, group := range profile.Syscalls {
		if group.Action == "" || group.Action == "SCMP_ACT_ALLOW" {
			names = append(names, group.Names...)
		}
	}
	return names, nil
}

// ApplyBaselineSeccompFilter instala o filtro BPF Seccomp base no processo do daemon.
func ApplyBaselineSeccompFilter() error {
	var profile seccompProfile
	if err := json.Unmarshal(baselineSeccompProfileJSON, &profile); err != nil {
		return fmt.Errorf("error parsing embedded baseline seccomp profile: %w", err)
	}

	errnoRet := profile.DefaultErrnoRet
	if errnoRet <= 0 {
		errnoRet = 1 // EPERM
	}

	filter, err := seccomp.NewFilter(seccomp.ActErrno.SetReturnCode(int16(errnoRet)))
	if err != nil {
		return fmt.Errorf("error creating baseline seccomp filter: %w", err)
	}
	defer filter.Release()

	// Desativa no_new_privs automático para permitir que o binário auxiliar ganhe file capabilities
	if err := filter.SetNoNewPrivsBit(false); err != nil {
		return fmt.Errorf("error disabling automatic NO_NEW_PRIVS on baseline filter: %w", err)
	}

	allowed := 0
	for _, group := range profile.Syscalls {
		action := seccomp.ActAllow
		if group.Action == "SCMP_ACT_ERRNO" {
			errno := group.ErrnoRet
			if errno <= 0 {
				errno = 1 // EPERM
			}
			action = seccomp.ActErrno.SetReturnCode(int16(errno))
		}

		for _, name := range group.Names {
			syscallID, err := seccomp.GetSyscallFromName(name)
			if err != nil {
				continue
			}
			if err := filter.AddRule(syscallID, action); err != nil {
				return fmt.Errorf("error setting rule for syscall %s: %w", name, err)
			}
			if action == seccomp.ActAllow {
				allowed++
			}
		}
	}
	if allowed == 0 {
		return fmt.Errorf("embedded baseline seccomp profile resolved to zero allowed syscalls")
	}

	return filter.Load()
}
