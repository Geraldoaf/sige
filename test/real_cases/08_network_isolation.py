import socket

print("=== [TEST 08] Network Isolation (CLONE_NEWNET) ===")
print("Action: Attempting outbound TCP connection to 8.8.8.8:53...")

try:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(3)
    s.connect(("8.8.8.8", 53))
    print("Result: UNEXPECTED - connection succeeded, network is NOT isolated!")
    s.close()
except OSError as e:
    print(f"Result: BLOCKED - {e}")

print("=== END TEST 08 ===")
