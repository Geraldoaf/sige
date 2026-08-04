print("=== [TEST 02] Memory Exhaustion (OOM Test) ===")
print("Action: Attempting to allocate >100MB memory in restricted 50MB sandbox...")

try:
    data = []
    for i in range(100):
        data.append(b"X" * (10 * 1024 * 1024))
    print("Result: UNEXPECTED - Memory allocation succeeded!")
except MemoryError:
    print("Result: BLOCKED - Memory limit enforced by Python runtime")

print("=== END TEST 02 ===")
