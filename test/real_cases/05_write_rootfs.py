print("=== [TEST 05] Unauthorized Rootfs Write Protection ===")
print("Action: Attempting to write malicious binary to system directory /bin/...")

try:
    with open("/bin/malicious_binary", "w") as f:
        f.write("test")
    print("Result: UNEXPECTED - Wrote to /bin!")
except PermissionError as e:
    print(f"Result: BLOCKED - System folder protected by Read-Only rootfs: {e}")
except Exception as e:
    print(f"Result: BLOCKED - Write attempt failed: {e}")

print("=== END TEST 05 ===")
