import time

print("=== [TEST 03] Infinite Loop Timeout ===")
print("Action: Entering infinite loop to trigger sandbox timeout limit...")

count = 0
while True:
    count += 1
    time.sleep(0.1)

print("=== END TEST 03 ===")
