import os
import sys

print("=== [TEST 29] Tmpfs Disk Space Exhaustion ===")
print("Action: Attempting to write files to /tmp until tmpfs quota is hit...")

chunk_size = 1024 * 1024  # 1MB chunks
total_written = 0
file_index = 0

try:
    while total_written < 200 * 1024 * 1024:  # Try writing up to 200MB (default limit is 64MB)
        path = f"/tmp/flood_{file_index}.bin"
        with open(path, "wb") as f:
            f.write(b"X" * chunk_size)
        total_written += chunk_size
        file_index += 1
    print(f"Result: UNEXPECTED - Managed to write {total_written // (1024*1024)}MB to /tmp without error!")
except OSError as e:
    print(f"Result: BLOCKED - Tmpfs limit enforced at {total_written // (1024*1024)}MB with error: {e}")

print("=== END TEST 29 ===")
