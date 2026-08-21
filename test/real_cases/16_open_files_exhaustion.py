print("=== [TEST 16] Open File Descriptor Limit (RLIMIT_NOFILE) ===")
print("Action: Opening files without closing until the limit is hit...")

handles = []
count = 0
try:
    while True:
        f = open(f"/tmp/fd_test_{count}.txt", "w")
        handles.append(f)
        count += 1
except OSError as e:
    print(f"Result: BLOCKED after opening {count} files - {e}")

print(f"Total file descriptors opened: {count}")
print("=== END TEST 16 ===")
