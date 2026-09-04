package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// KeyProvider define o contrato para resolução e validação de chaves de API.
type KeyProvider interface {
	ResolveKey() string
	Validate(providedKey string) bool
	IsGenerated() bool
}

// FileKeyProvider implementa KeyProvider com resolução em cascata:
// 1. Docker Secret (/run/secrets/sige_api_key)
// 2. Variável de ambiente SIGE_API_KEY
// 3. Fallback unauthenticated se SIGE_ALLOW_UNAUTHENTICATED=true
// 4. Chave persistida em disco (<stateDir>/api_key)
// 5. Geração aleatória de 32 bytes criptograficamente seguros (crypto/rand)
type FileKeyProvider struct {
	stateDir             string
	allowUnauthenticated bool
	resolvedKey          string
	isGenerated          bool
	once                 sync.Once
}

func NewFileKeyProvider(stateDir string, allowUnauthenticated bool) *FileKeyProvider {
	if stateDir == "" {
		stateDir = "/var/lib/sige"
	}
	return &FileKeyProvider{
		stateDir:             stateDir,
		allowUnauthenticated: allowUnauthenticated,
	}
}

func (p *FileKeyProvider) ResolveKey() string {
	p.once.Do(func() {
		// 1. Docker secret
		if secret, err := os.ReadFile("/run/secrets/sige_api_key"); err == nil {
			if k := strings.TrimSpace(string(secret)); k != "" {
				p.resolvedKey = k
				return
			}
		}

		// 2. Variável de ambiente
		if envKey := strings.TrimSpace(os.Getenv("SIGE_API_KEY")); envKey != "" {
			p.resolvedKey = envKey
			return
		}

		// 3. Desenvolvimento sem autenticação
		if p.allowUnauthenticated || os.Getenv("SIGE_ALLOW_UNAUTHENTICATED") == "true" {
			return
		}

		// 4 & 5. Arquivo persistido ou geração criptográfica
		p.resolvedKey = p.loadOrGenerate()
	})
	return p.resolvedKey
}

func (p *FileKeyProvider) loadOrGenerate() string {
	path := filepath.Join(p.stateDir, "api_key")

	if data, err := os.ReadFile(path); err == nil {
		if k := strings.TrimSpace(string(data)); k != "" {
			return k
		}
	}

	p.isGenerated = true
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("falha crítica de entropia ao gerar API Key: %v", err))
	}
	key := hex.EncodeToString(buf)

	if err := os.MkdirAll(p.stateDir, 0700); err == nil {
		_ = os.WriteFile(path, []byte(key+"\n"), 0600)
	}

	return key
}

func (p *FileKeyProvider) Validate(providedKey string) bool {
	expected := p.ResolveKey()
	if expected == "" && (p.allowUnauthenticated || os.Getenv("SIGE_ALLOW_UNAUTHENTICATED") == "true") {
		return true
	}
	if expected == "" || providedKey == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(providedKey)) == 1
}

func (p *FileKeyProvider) IsGenerated() bool {
	p.ResolveKey()
	return p.isGenerated
}
