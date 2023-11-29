import os
import time
import csv
import logging

logging.basicConfig(format='%(asctime)s %(levelname)s:%(message)s', level=logging.INFO)

import argparse
import numpy as np

from itertools import product
from typing import List

parser = argparse.ArgumentParser()
parser.add_argument('--out-dir', type=str, default='/osdata/osgroup4/download_imgs')
args = parser.parse_args()

top_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
tmp_dir = bin_dir = os.path.join(top_dir, 'temp')
data_dir = os.path.join(tmp_dir, 'data')

client22 = os.path.join(bin_dir, 'client22')
assert os.path.exists(client22)

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


def run_experiment(hosts: List[str], threads_per_server: int, batch_size: int,
                   stats_file: str, out_dir: str) -> float:
    assert os.path.exists(out_dir)
    for f in os.listdir(out_dir):
        os.remove(os.path.join(out_dir, f))

    cmd = f'{client22} -n-t {threads_per_server} -out-dir {out_dir} -stats-file {stats_file} -batch-size {batch_size} '
    assert len(hosts) > 0, 'no hosts provided'
    cmd += ' -host=' + ' -host='.join(hosts)
    cmd += ' -n-s ' + str(len(hosts))
    logging.info(f'Running experiment with {len(hosts)} hosts, {threads_per_server} threads per server and batch size {batch_size}.')
    start_time = time.time()
    os.system(cmd)
    spend_time = time.time() - start_time
    logging.info(f'Experiment with {len(hosts)} hosts, {threads_per_server} threads per server and batch size {batch_size} ended in {spend_time} seconds.')
    with open(stats_file, 'a') as f:
        f.write(f'{spend_time}\n')
    return spend_time

batch_size = [1, 2, 4, 8, 16, 25, 40]
threads_per_server = [1, 2, 4, 8, 10, 12, 16]

for b, t in product(batch_size, threads_per_server):
    run_experiment(hosts=[ips[18][0]], threads_per_server=t, batch_size=b,
                   stats_file=os.path.join(data_dir, f'batch{b}_threads{t}_host{1}.csv'),
                   out_dir=args.out_dir)
    run_experiment(hosts=[ips[17][0], ips[16][0]], threads_per_server=t, batch_size=b,
                   stats_file=os.path.join(data_dir, f'batch{b}_threads{t}_host{2}.csv'),
                   out_dir=args.out_dir)
    run_experiment(hosts=[ips[15][0], ips[14][0], ips[13][0], ips[12][0]], threads_per_server=t, batch_size=b,
                   stats_file=os.path.join(data_dir, f'batch{b}_threads{t}_host{4}.csv'),
                   out_dir=args.out_dir)
    run_experiment(hosts=[ips[11][0], ips[10][0], ips[9][0], ips[8][0], ips[7][0], ips[6][0]], threads_per_server=t, batch_size=b,
                   stats_file=os.path.join(data_dir, f'batch{b}_threads{t}_host{6}.csv'),
                   out_dir=args.out_dir)

