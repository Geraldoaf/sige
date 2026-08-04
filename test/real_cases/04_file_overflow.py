print("=== [TEST 04] File Size Limit Overflow ===")
print("Action: Attempting to write 50MB file exceeding 15MB RLIMIT_FSIZE limit...")

try:
    with open("/tmp/overflow.txt", "wb") as f:
        chunk = b"A" * (1024 * 1024)
        for i in range(50):
            f.write(chunk)
            f.flush()
    print("Result: UNEXPECTED - File writing succeeded!")
except OSError as e:
    print(f"Result: BLOCKED - File writing stopped by system limit: {e}")

print("=== END TEST 04 ===")
