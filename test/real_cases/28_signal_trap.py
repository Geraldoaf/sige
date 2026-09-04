import signal
import sys
import time

print("=== [TEST 28] Signal Trapping & Unkillable Process ===")
print("Action: Trapping all catchable signals and entering infinite loop...")

def handle_signal(signum, frame):
    print(f"Trapped signal {signum}, ignoring it and continuing execution!")

signals_to_trap = [
    signal.SIGTERM,
    signal.SIGINT,
    signal.SIGHUP,
    signal.SIGQUIT,
    signal.SIGUSR1,
    signal.SIGUSR2,
    signal.SIGALRM,
]

for sig in signals_to_trap:
    try:
        signal.signal(sig, handle_signal)
    except (OSError, ValueError, RuntimeError):
        pass

print("Action: Signal traps installed. Spinning indefinitely (testing cgroup.kill / SIGKILL timeout)...")
sys.stdout.flush()

counter = 0
while True:
    counter += 1
    time.sleep(0.01)
