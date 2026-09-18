"""Multi-port TCP forwarder: listen_port -> target_port on 127.0.0.1.
Usage: python pg-port-forward.py [LISTEN_PORT TARGET_PORT] ...
Defaults: 5432 -> 5433 (postgres), 6379 -> 6380 (valkey), 4222 -> 4223 (nats)
"""
import socket
import threading
import sys

PAIRS = [
    (5432, 5433),   # postgres
    (6379, 6380),   # valkey
    (4222, 4223),   # nats
]

if len(sys.argv) > 1:
    PAIRS = []
    for i in range(1, len(sys.argv), 2):
        PAIRS.append((int(sys.argv[i]), int(sys.argv[i+1])))

def pipe(src, dst):
    try:
        while True:
            data = src.recv(4096)
            if not data:
                break
            dst.sendall(data)
    except Exception:
        pass
    finally:
        try: src.close()
        except: pass
        try: dst.close()
        except: pass

def handle(client, target_addr):
    try:
        target = socket.create_connection(target_addr, timeout=5)
    except Exception as e:
        print(f"target connect {target_addr} failed: {e}", file=sys.stderr)
        client.close()
        return
    threading.Thread(target=pipe, args=(client, target), daemon=True).start()
    threading.Thread(target=pipe, args=(target, client), daemon=True).start()

def serve(listen_port, target_port):
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', listen_port))
    server.listen(50)
    target_addr = ('127.0.0.1', target_port)
    print(f"forwarding 127.0.0.1:{listen_port} -> 127.0.0.1:{target_port}", flush=True)
    while True:
        client, addr = server.accept()
        threading.Thread(target=handle, args=(client, target_addr), daemon=True).start()

threads = []
for lp, tp in PAIRS:
    t = threading.Thread(target=serve, args=(lp, tp), daemon=True)
    t.start()
    threads.append(t)

print(f"started {len(PAIRS)} port forwarders", flush=True)
import time
while True:
    time.sleep(60)
