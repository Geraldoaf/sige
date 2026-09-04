import sys

print("=== [TEST 36] Binary / Null Byte / UTF-8 Corruption Stream ===")
print("Action: Emitting raw binary null bytes and invalid UTF-8 bytes to stdout and stderr...")

# Print raw null bytes and non-printable sequences
sys.stdout.buffer.write(b"BINARY_START\x00\x01\x02\xff\xfe\x00\x00\xaa\xbbBINARY_END\n")
sys.stdout.flush()

sys.stderr.buffer.write(b"STDERR_NULL\x00\xff\xeeSTDERR_END\n")
sys.stderr.flush()

print("Result: Completed stream emit")
print("=== END TEST 36 ===")
