import os
import time

print("=== [TEST 07] Fork Bomb (cgroup pids.max) ===")
print("Action: Forking repeatedly, keeping children alive, to trigger the PID limit...")

count = 0
try:
    while True:
        pid = os.fork()
        if pid == 0:
            time.sleep(5)
            os._exit(0)
        count += 1
except OSError as e:
    print(f"Result: BLOCKED after {count} forks - {e}")
else:
    print(f"Result: UNEXPECTED - loop exited normally after {count} forks")

print(f"Total forks attempted before stopping: {count}")
print("=== END TEST 07 ===")
