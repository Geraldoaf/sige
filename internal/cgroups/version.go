package cgroups

import (
	"github.com/containerd/cgroups/v3"
)

// VerifyCgroupsVersion verifica se o sistema está operando com cgroups v2 no modo Unified.
func VerifyCgroupsVersion() bool {
	return cgroups.Mode() == cgroups.Unified
}

// GetCgroupsMode retorna o modo atual do cgroups no sistema operacional.
func GetCgroupsMode() string {
	switch cgroups.Mode() {
	case cgroups.Unified:
		return "v2 (Unified)"
	case cgroups.Legacy:
		return "v1 (Legacy)"
	case cgroups.Hybrid:
		return "Hybrid (v1/v2)"
	default:
		return "Unavailable"
	}
}
