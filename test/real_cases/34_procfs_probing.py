import os

print("=== [TEST 34] Procfs Sensitive Entries Probing ===")
print("Action: Checking access to sensitive /proc files...")

sensitive_paths = [
    "/proc/kcore",
    "/proc/kallsyms",
    "/proc/sysrq-trigger",
    "/proc/acpi",
    "/proc/sys/kernel/core_pattern",
    "/proc/sched_debug",
    "/proc/1/environ",
    "/proc/1/cmdline",
]

for p in sensitive_paths:
    try:
        with open(p, "rb") as f:
            data = f.read(16)
        print(f"Result: SENSITIVE_READ - Successfully read {len(data)} bytes from {p}")
    except OSError as e:
        print(f"OK - {p} is inaccessible ({type(e).__name__})")

print("=== END TEST 34 ===")
