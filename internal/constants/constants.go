package constants

const (
	DefaultUIDNobody = 65534
	DefaultGIDNobody = 65534

	DefaultPIDLimit = 50

	CgroupPrefix = "/sige/"

	DefaultCPUPeriod = 100000

	OldRootPerms = 0777
	RootfsPerms  = 0755

	BytesInMB = 1024 * 1024

	// Valores padrão de fallback
	DefaultFallbackTimeoutSec = 5
	DefaultFallbackCPUPercent = 50

	// Tetos máximos de recursos da API
	DefaultCeilingMemoryMB      = 512
	DefaultCeilingCPUPercent    = 100
	DefaultCeilingTimeoutSec    = 30
	DefaultCeilingTmpLimitMB    = 256
	DefaultCeilingMaxFileSizeMB = 64
	DefaultCeilingMaxOpenFiles  = 512

	// Limite de captura de saída stdout/stderr (1MB)
	DefaultMaxOutputBytes = 1 * 1024 * 1024

	// Limites de recursos para compilação C/C++
	DefaultCompileMemoryMB     = 256
	DefaultCompileCPUPercent   = 100
	DefaultCompileTimeoutSec   = 10
	DefaultCompileTmpLimitMB   = 64
	DefaultCompileMaxFileMB    = 64
	DefaultCompileMaxOpenFiles = 256
)
