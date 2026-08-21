import multiprocessing
import os
import queue
import threading

print("=== [TEST 26] Legitimate Concurrency Primitives ===")
print("Action: Exercising threads, processes and POSIX semaphores...")

# Este caso existe por causa de uma regressão real: ao negar clone3 com
# ActKillProcess, QUALQUER criação de thread passou a matar o processo sem
# mensagem — a glibc 2.34+ usa clone3 no pthread_create e só cai para clone()
# se receber ENOSYS. Matar não dá chance ao fallback. A syscall continua
# negada; o que mudou foi a forma de negar (ver denyClone3WithENOSYS em
# internal/sandbox/seccomp.go).
q = queue.Queue()
t = threading.Thread(target=lambda: q.put("ok"))
t.start()
t.join()
print("OK - threading:", q.get())

# multiprocessing.Lock depende de semáforo POSIX, que exige /dev/shm.
lock = multiprocessing.Lock()
with lock:
    print("OK - multiprocessing.Lock (precisa de /dev/shm)")

with multiprocessing.Pool(2) as pool:
    print("OK - multiprocessing.Pool:", pool.map(abs, [-3, -1, -2]))

pid = os.fork()
if pid == 0:
    os._exit(7)
_, status = os.waitpid(pid, 0)
print("OK - os.fork + waitpid, exit do filho:", os.WEXITSTATUS(status))

print("OK - /dev/shm presente:", os.path.isdir("/dev/shm"))

print("=== END TEST 26 ===")
