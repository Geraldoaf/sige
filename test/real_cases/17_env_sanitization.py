import os

print("=== [TEST 17] Environment Sanitization ===")
print("Action: Dumping every environment variable visible to the sandboxed process...")

for key, value in sorted(os.environ.items()):
    print(f"{key}={value}")

print(f"Total variables: {len(os.environ)}")
print("Expected: only PATH, HOME, LANG, TERM (ver cleanEnv em cmd/internal_launch.go) —")
print("nada do processo da API (ex.: SIGE_API_KEY, TCC_EXECUTABLE, SIGE_TRUSTED_PROXIES,")
print("SIGE_WORKSPACE_DIR não devem aparecer aqui).")
print("=== END TEST 17 ===")
