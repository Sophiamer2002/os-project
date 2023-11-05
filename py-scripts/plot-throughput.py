import os
import sys

import argparse
from matplotlib import pyplot as plt

parser = argparse.ArgumentParser()
parser.add_argument('--data-dir', type=str, help='Directory containing the data files')
parser.add_argument('--out-dir', type=str, help='Directory to save the plots')
parser.add_argument('--threads', type=int, default=6, help='The maximum number of threads')
parser.add_argument('--log-cap', type=int, default=5, help='The maximum log(capacity) of the queue')

args = parser.parse_args()

data_file_template = "cap_{}_threads_{}.txt"

data = [[] for _ in range(args.log_cap + 1)]
for log_cap in range(args.log_cap + 1):
    for threads in range(1, args.threads + 1):
        data_file = os.path.join(args.data_dir, data_file_template.format(2 ** log_cap, threads))
        with open(data_file, 'r') as f:
            # read the first two integers
            _in, _out = map(int, f.readline().split())
            data[log_cap].append((_in + _out) / 2)

# plot the data
plt.figure(figsize=(20, 10))
plt.xlabel('Number of threads')
plt.ylabel('Throughput (10 milliseconds)')
plt.title('Throughput vs Number of threads')

for log_cap in range(args.log_cap + 1):
    plt.plot(range(1, args.threads + 1), data[log_cap], label='Capacity: %d' % (2 ** log_cap), marker='o')

plt.legend()
plt.savefig(os.path.join(args.out_dir, 'part11_throughput_fig.png'))
