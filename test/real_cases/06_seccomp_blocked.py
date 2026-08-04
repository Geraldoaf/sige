import ctypes

print("=== [TEST 06] Seccomp Forbidden Syscall Filter ===")
print("Action: Executing forbidden syscall unshare via libc...")

try:
    libc = ctypes.CDLL("libc.so.6")
    res = libc.unshare(0x10000000)
    print(f"Result: Syscall returned code {res}")
except Exception as e:
    print(f"Result: BLOCKED - Syscall intercepted by Seccomp: {e}")

print("=== END TEST 06 ===")
