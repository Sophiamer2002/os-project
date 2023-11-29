import os
import time
import csv
import logging

logging.basicConfig(format='%(asctime)s %(levelname)s:%(message)s', level=logging.INFO)

from concurrent import futures

import argparse
import numpy as np

parser = argparse.ArgumentParser()
parser.add_argument('--out-dir', type=str, default='/osdata/osgroup4/download_imgs')
args = parser.parse_args()

top_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
tmp_dir = bin_dir = os.path.join(top_dir, 'temp')
data_dir = os.path.join(tmp_dir, 'data')

client21 = os.path.join(bin_dir, 'client21')
assert os.path.exists(client21)

ips = [("10.1.0.91:51151", 1),   # threads = 1
       ("10.1.0.92:51151", 2),   # threads = 2
       ("10.1.0.93:51151", 4),   # threads = 4
       ("10.1.0.94:51151", 8),   # threads = 8
       ("10.1.0.95:51151", 12),   # threads = 12
       ("10.1.0.96:51151", 24),   # threads = 24
       ("10.1.0.98:51151", 24),   # threads = 24
       ("10.1.0.99:51151", 24),   # threads = 24
       ("10.1.0.116:51151", 24),   # threads = 24
       ("10.1.0.102:51151", 24),   # threads = 24
       ("10.1.0.103:51151", 24),   # threads = 24
       ("10.1.0.104:51151", 24),   # threads = 24
       ("10.1.0.105:51151", 24),   # threads = 24
       ("10.1.0.107:51151", 24),   # threads = 24
       ("10.1.0.109:51151", 24),   # threads = 24
       ("10.1.0.110:51151", 24),   # threads = 24
       ("10.1.0.111:51151", 24),   # threads = 24
       ("10.1.0.112:51151", 24),   # threads = 24
       ("10.1.0.113:51151", 24),   # threads = 24
       "10.1.0.115:51151"]

def run_client_cmd(client: str, addr: str, threads: int, stats_file: str,
                   time_file:str, out_dir: str, proc_id=0, master=False) -> str:
    cmd = f'{client} -n-t {threads} -addr {addr} -out-dir {out_dir} -stats-file {stats_file}'
    cmd += f' -time-file {time_file}'
    if master: cmd += ' -master'
    else: cmd += f' -proc-id {proc_id}'
    return cmd

def run_experiment(addr, n_threads=[1],
                   stats_file=os.path.join(data_dir, 'stats.csv'),
                   time_file=os.path.join(data_dir, 'time.txt')):
    if os.path.exists(os.path.join('/dev/shm', 'shm_atomic_int')):
        os.remove(os.path.join('/dev/shm', 'shm_atomic_int'))
    for f in os.listdir(args.out_dir):
        os.remove(os.path.join(args.out_dir, f))
    
    n_clients = len(n_threads)
    assert n_clients > 0, 'n_clients must be positive'

    cmds = []
    logging.info(f'Running experiment {n_threads} on server {addr}...')

    with futures.ThreadPoolExecutor() as excute:
        # run non-master clients first
        for i in range(1, n_clients):
            cmd = run_client_cmd(client21, addr, threads=n_threads[i], out_dir=args.out_dir,
                                 proc_id=i, stats_file=stats_file, time_file=time_file)
            cmds.append(excute.submit(os.system, cmd))
        
        # then start master client, so that they can start together
        cmd = run_client_cmd(client21, addr, threads=n_threads[0], out_dir=args.out_dir,
                             stats_file=stats_file, master=True, time_file=time_file)
        cmds.append(excute.submit(os.system, cmd))
        start_time = time.time()

    end_time = time.time()
    elapsed_time = end_time - start_time
    logging.info(f'Experiment {n_threads} finished in {elapsed_time:.2f}s.')
    return elapsed_time, read_stats(stats_file)

def read_stats(path: str) -> dict:
    with open(path, 'r') as f:
        reader = csv.reader(f)
        transposed = list(zip(*reader))
        return {
            t[0]: nparray(t[1:])
            for t in transposed
        }

def nparray(data: list) -> np.ndarray:
    try:
        return np.array(data, dtype=np.float64)
    except ValueError:
        return np.array(data)

stats1x8, time1x8 = run_experiment(
    ips[10][0], [8],
    stats_file=os.path.join(data_dir, 'stats1x8.csv'),
    time_file=os.path.join(data_dir, 'time1x8.txt'))
stats2x4, time2x4 = run_experiment(
    ips[10][0], [4, 4],
    stats_file=os.path.join(data_dir, 'stats2x4.csv'),
    time_file=os.path.join(data_dir, 'time2x4.txt'))
stats4x2, time4x2 = run_experiment(
    ips[10][0], [2, 2, 2, 2],
    stats_file=os.path.join(data_dir, 'stats4x2.csv'),
    time_file=os.path.join(data_dir, 'time4x2.txt'))
stats8x1, time8x1 = run_experiment(
    ips[10][0], [1, 1, 1, 1, 1, 1, 1, 1],
    stats_file=os.path.join(data_dir, 'stats8x1.csv'),
    time_file=os.path.join(data_dir, 'time8x1.txt'))

for i in range(6):
    stats1, time1 = run_experiment(
        ips[i][0], [1],
        stats_file=os.path.join(data_dir, f'stats1_serverthreads{ips[i][1]}.csv'),
        time_file=os.path.join(data_dir, f'time1_serverthreads{ips[i][1]}.txt'))
    stats2, time2 = run_experiment(
        ips[i][0], [2],
        stats_file=os.path.join(data_dir, f'stats2_serverthreads{ips[i][1]}.csv'),
        time_file=os.path.join(data_dir, f'time2_serverthreads{ips[i][1]}.txt'))
    stats4, time4 = run_experiment(
        ips[i][0], [4],
        stats_file=os.path.join(data_dir, f'stats4_serverthreads{ips[i][1]}.csv'),
        time_file=os.path.join(data_dir, f'time4_serverthreads{ips[i][1]}.txt'))
    stats8, time8 = run_experiment(
        ips[i][0], [8],
        stats_file=os.path.join(data_dir, f'stats8_serverthreads{ips[i][1]}.csv'),
        time_file=os.path.join(data_dir, f'time8_serverthreads{ips[i][1]}.txt'))
    stats12, time12 = run_experiment(
        ips[i][0], [12],
        stats_file=os.path.join(data_dir, f'stats12_serverthreads{ips[i][1]}.csv'),
        time_file=os.path.join(data_dir, f'time12_serverthreads{ips[i][1]}.txt'))
    stats16, time16 = run_experiment(
        ips[i][0], [16],
        stats_file=os.path.join(data_dir, f'stats16_serverthreads{ips[i][1]}.csv'),
        time_file=os.path.join(data_dir, f'time16_serverthreads{ips[i][1]}.txt'))
