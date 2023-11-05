import os
import sys

import argparse
from matplotlib import pyplot as plt

parser = argparse.ArgumentParser()
parser.add_argument('--data-file', type=str, help='The path of the data file')
parser.add_argument('--out-file', type=str, help='The path of the output file')
parser.add_argument('--threads', type=int, help='The number of threads')
parser.add_argument('--cap', type=int, help='The capacity of the queue')

args = parser.parse_args()

data_file_name = "cap_{}_threads_{}.txt".format(args.cap, args.threads)
assert args.data_file.endswith(data_file_name)

Max_latency = 10000

data_file = args.data_file
with open(data_file, 'r') as f:
    # read the first two integers
    _in, _out = map(int, f.readline().split())
    # the second line
    sec_line = list(f.readline().strip().split())
    # the third line
    third_line = list(f.readline().strip().split())  

    pro_dis_in = [0 for _ in range(Max_latency)]
    pro_dis_out = [0 for _ in range(Max_latency)]
    cul_dis_in = [0.0 for _ in range(Max_latency)]
    cul_dis_out = [0.0 for _ in range(Max_latency)]
    sum_in = 0
    sum_out = 0

    # get the probability
    for i in range(Max_latency):
        if i < len(sec_line):
            id_in = int(sec_line[i])
            if id_in < Max_latency:
                pro_dis_in[id_in] += 1
        if i < len(third_line):
            id_out = int(third_line[i])
            if id_out < Max_latency:
                pro_dis_out[id_out] += 1
    
    # get the culmulative probability
    for i in range(Max_latency):
        sum_in += pro_dis_in[i]
        sum_out += pro_dis_out[i]
        cul_dis_in[i] = float(sum_in/_in)
        cul_dis_out[i] = float(sum_out/_out)
        

    # plot the data
    plt.figure(figsize=(20, 10))
    plt.xlabel('Latency')
    plt.ylabel('the culmulative probability')
    plt.ylim(0,1)
    plt.title('The cdf of the enqueue/dequeue operation')
    plt.plot(range(Max_latency), cul_dis_in, label='in', marker='o')
    plt.plot(range(Max_latency), cul_dis_out, label='out', marker='o')
    plt.legend()
    plt.savefig(args.out_file)
