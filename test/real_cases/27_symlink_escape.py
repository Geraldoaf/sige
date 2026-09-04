import os
import sys

print("=== [TEST 27] Symlink & Directory Traversal Escape ===")
print("Action: Attempting to create symlinks to host paths and break out of sandbox...")

escapes = [
    ("/tmp/link_passwd", "../../../../etc/passwd"),
    ("/tmp/link_shadow", "../../../../etc/shadow"),
    ("/tmp/link_root", "/proc/1/root/etc/passwd"),
    ("/tmp/link_cwd", "/proc/1/cwd"),
    ("/tmp/link_cgroup", "/sys/fs/cgroup"),
]

for link_name, target in escapes:
    try:
        if os.path.exists(link_name):
            os.remove(link_name)
        os.symlink(target, link_name)
        # Try reading through symlink
        with open(link_name, "rb") as f:
            data = f.read(32)
        print(f"Result: UNEXPECTED - Read {len(data)} bytes from {target} via symlink {link_name}")
    except OSError as e:
        print(f"OK - Symlink {link_name} -> {target} blocked/unreadable ({type(e).__name__})")

print("=== END TEST 27 ===")
