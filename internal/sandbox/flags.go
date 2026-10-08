package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// CompileFlags suporta deserialização flexível a partir de um array JSON ([]string) ou de uma string única com flags separadas por espaço.
type CompileFlags []string

func (c *CompileFlags) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*c = nil
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		*c = list
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*c = ParseFlagsString(str)
		return nil
	}
	return errors.New("must be a string or an array of strings")
}

// ParseFlagsString divide uma string de flags em tokens respeitando aspas simples e duplas.
func ParseFlagsString(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var flags []string
	var current strings.Builder
	inQuote := false
	var quoteChar rune

	for _, r := range s {
		switch {
		case inQuote:
			if r == quoteChar {
				inQuote = false
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			inQuote = true
			quoteChar = r
		case unicode.IsSpace(r):
			if current.Len() > 0 {
				flags = append(flags, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		flags = append(flags, current.String())
	}
	return flags
}

// ValidateCompileFlags valida as flags de compilação fornecidas para linguagens compiladas.
func ValidateCompileFlags(language string, flags []string) error {
	if len(flags) == 0 {
		return nil
	}
	lang := strings.ToLower(strings.TrimSpace(language))
	if lang != "c" && lang != "cpp" && lang != "c++" {
		return errors.New("Field 'compile_flags' is only supported for compiled languages (c, cpp)")
	}
	if len(flags) > 50 {
		return errors.New("Too many flags in 'compile_flags': maximum allowed is 50")
	}
	for i, f := range flags {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if len(f) > 255 {
			return fmt.Errorf("Flag in 'compile_flags[%d]' is too long: maximum allowed is 255 bytes", i)
		}
		if strings.ContainsAny(f, "\x00\n\r") {
			return fmt.Errorf("Flag in 'compile_flags[%d]' contains invalid control characters", i)
		}
		if !strings.HasPrefix(f, "-") || f == "-" || f == "--" {
			return fmt.Errorf("Flag in 'compile_flags[%d]' ('%s') is invalid: flags must start with '-'", i, f)
		}
		// Não permite sobrescrever binário de saída (-o, --output)
		if f == "-o" || strings.HasPrefix(f, "--output") || strings.HasPrefix(f, "-o=") || (len(f) > 2 && f[0] == '-' && f[1] == 'o') {
			return errors.New("Field 'compile_flags' cannot override output binary (-o, --output)")
		}
		// Não permite flags que impedem a geração do executável final
		if f == "-c" || f == "-S" || f == "-E" || strings.HasPrefix(f, "-c=") || strings.HasPrefix(f, "-S=") || strings.HasPrefix(f, "-E=") {
			return fmt.Errorf("Field 'compile_flags' cannot include '%s' (compilation must produce an executable binary)", f)
		}
		// Bloqueia flags perigosas do GCC/G++ (plugins, specs customizados, wrappers, inclusão forçada de arquivos do sistema ou gravação de arquivos auxiliares)
		if strings.HasPrefix(f, "-fplugin") ||
			strings.HasPrefix(f, "-specs") || strings.HasPrefix(f, "--specs") ||
			f == "-wrapper" || strings.HasPrefix(f, "-wrapper=") ||
			f == "-B" || strings.HasPrefix(f, "-B") ||
			f == "-include" || strings.HasPrefix(f, "-include=") ||
			f == "-imacros" || strings.HasPrefix(f, "-imacros=") ||
			f == "-MF" || strings.HasPrefix(f, "-MF=") || f == "-MD" || f == "-MMD" ||
			strings.HasPrefix(f, "-save-temps") || strings.HasPrefix(f, "-dumpbase") || strings.HasPrefix(f, "-dumpdir") ||
			f == "-Xlinker" || f == "-Xpreprocessor" || f == "-Xassembler" ||
			strings.HasPrefix(f, "-Wl,-o") || strings.HasPrefix(f, "-Wl,--output") {
			return fmt.Errorf("Field 'compile_flags' contains restricted compiler option '%s'", f)
		}
	}
	return nil
}
