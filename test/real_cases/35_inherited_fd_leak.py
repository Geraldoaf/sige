import os
import stat

print("=== [TEST 35] Inherited File Descriptors Leak Check ===")
print("Action: Probing open file descriptors inherited across execve...")

leaked_count = 0
# Standard fds are 0 (stdin), 1 (stdout), 2 (stderr)
for fd in range(3, 100):
    try:
        st = os.fstat(fd)
        mode = "unknown"
        if stat.S_ISREG(st.st_mode):
            mode = "file"
        elif stat.S_ISFIFO(st.st_mode):
            mode = "pipe"
        elif stat.S_ISSOCK(st.st_mode):
            mode = "socket"
        elif stat.S_ISCHR(st.st_mode):
            mode = "char_device"

        print(f"Result: UNEXPECTED - Found open inherited fd={fd} (type={mode}, inode={st.st_ino})")
        leaked_count += 1
    except OSError:
        pass

if leaked_count == 0:
    print("Result: SUCCESS - No leaked file descriptors found (fds 3-100 closed)")

print("=== END TEST 35 ===")
