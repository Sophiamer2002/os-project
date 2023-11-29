import os
import csv
import logging
from itertools import product

logging.basicConfig(format='%(asctime)s %(levelname)s:%(message)s', level=logging.INFO)

import numpy as np
import matplotlib.pyplot as plt

top_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
tmp_dir = bin_dir = os.path.join(top_dir, 'temp')
fig_dir = os.path.join(tmp_dir, 'fig')
data_dir = os.path.join(tmp_dir, 'data')

def read_stats(b:int, t: int, h: int) -> dict:
    path = os.path.join(data_dir, f'batch{b}_threads{t}_host{h}.csv')
    with open(path, 'r') as f:
        reader = list(csv.reader(f))
        # extract the last line, which records the total time
        last_line = reader[-1]
        transposed = list(zip(*(reader[:-1])))
        a =  {
            t[0]: nparray(t[1:])
            for t in transposed
        }
        a['total_time'] = float(last_line[0]) * (10 ** 9) # in nanoseconds
        a['throughput'] = 2000 * (10**9) / a['total_time']
        a['threads_per_server'] = t
        a['hosts'] = h
        a['batch_size'] = b
    
    return a

def nparray(data: list) -> np.ndarray:
    try:
        return np.array(data, dtype=np.float64)
    except ValueError:
        return np.array(data)
    
batch_size = [1, 2, 4, 8, 16, 25, 40]
threads_per_server = [1, 2, 4, 8, 10, 12, 16]
hosts = [1, 2, 4, 6]

part22_stats = {
    (b, t, h): read_stats(b, t, h)
    for b, t, h in product(batch_size, threads_per_server, hosts)
}

def get_stats(server_thread: int, client_thread: int) -> dict:
    def get_total_time(path: str) -> float:
        with open(path, 'r') as f:
            times = [float(x.strip()) for x in f.readlines()]
            return sum(times) / len(times)
    def read_stats(path: str) -> dict:
        with open(path, 'r') as f:
            reader = csv.reader(f)
            transposed = list(zip(*reader))
            return {
                t[0]: nparray(t[1:])
                for t in transposed
            }
    stats_file = os.path.join(data_dir, f'stats{client_thread}_serverthreads{server_thread}.csv')
    time_file = os.path.join(data_dir, f'time{client_thread}_serverthreads{server_thread}.txt')
    a = read_stats(stats_file)
    a['total_time'] = get_total_time(time_file)
    a['throughput'] = 2000 * (10**9) / a['total_time']  # time is measured in nanoseconds
    a['server_thread'] = server_thread
    a['client_thread'] = client_thread
    return a

server_threads = [1, 2, 4, 8, 12, 24]
client_threads = [1, 2, 4, 8, 12, 16]

part21_stats = {
    (server_thread, client_thread): get_stats(server_thread, client_thread)
    for server_thread in server_threads
    for client_thread in client_threads
}

#region 
#plot set 1: compare performance vs. number of client threads
#            with different batch sizes in the plot

def plot_something1(func, title, x_label, y_label, filename):
    data = [
        [
            func(part22_stats[(b, t, 1)])
            for t in threads_per_server
        ]
        for b in batch_size
    ]

    data21 = [func(part21_stats[(24, t)]) for t in client_threads]
    fig, ax = plt.subplots()
    for i in range(len(batch_size)):
        ax.plot(threads_per_server, data[i], label=f'{batch_size[i]} batch size')
    ax.plot(client_threads, data21, label='baseline: GetSingleImg')

    ax.set_title(title)
    ax.set_xlabel(x_label)
    ax.set_ylabel(y_label)
    ax.legend()
    plt.savefig(os.path.join(fig_dir, filename))

plot_something1(lambda x: x['throughput'], 'Throughput vs. threads per server', 'threads per server', 'throughput (images per second)', 'part22-throughput1.png')
plot_something1(
    lambda x: x['total_latency'].mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Latency vs. threads per server', 'threads per server', 'latency (milliseconds)', 'part22-latency1.png'
)
plot_something1(
    lambda x: x['total_latency'].std() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'latency standard deviation vs. threads per server', 'threads per server', 'latency standard deviation (milliseconds)', 'part22-latency_std1.png'
)
plot_something1(
    lambda x: (x['rtt_latency'] - x['inserver_latency']).mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Network latency vs. threads per server', 'threads per server', 'network latency (milliseconds)', 'part22-network_latency1.png'
)
plot_something1(
    lambda x: (x['rtt_latency'] - x['inserver_latency']).std() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Network latency standard deviation vs. threads per server', 'threads per server', 'network latency standard deviation (milliseconds)', 'part22-network_latency_std1.png'
)
plot_something1(
    lambda x: (x['inserver_latency'] - x['handle_latency']).mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Queue latency vs. threads per server', 'threads per server', 'queue latency (milliseconds)', 'part22-queue_latency1.png'
)
plot_something1(
    lambda x: x['ongoing_tasks'].mean(),
    'Queue length vs. threads per server', 'threads per server', 'queue length', 'part22-queue_length1.png'
)

plt.cla()
plt.close("all")

#endregion

#region
#plot set 2: performance vs. number of hosts, batch size fixed at 1
#            with different threads per server in the plot

def plot_something2(func, title, x_label, y_label, filename):
    data = [
        [
            func(part22_stats[(1, t, h)])
            for t in threads_per_server
        ]
        for h in hosts
    ]

    fig, ax = plt.subplots()
    for i in range(len(hosts)):
        ax.plot(threads_per_server, data[i], label=f'{hosts[i]} host(s)')

    ax.set_title(title)
    ax.set_xlabel(x_label)
    ax.set_ylabel(y_label)
    ax.legend()
    plt.savefig(os.path.join(fig_dir, filename))

plot_something2(
    lambda x: x['throughput'],
    'Throughput vs. threads per server', 'threads per server', 'throughput (images per second)', 'part22-throughput2.png'
)
plot_something2(
    lambda x: x['throughput'] / part22_stats[(1, 1, 1)]['throughput'] / x['hosts'],
    'Speed up/number of hosts vs. threads per server', 'threads per server', 'speed up', 'part22-speed_up2.png'
)
plot_something2(
    lambda x: x['total_latency'].mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Latency vs. threads per server', 'threads per server', 'latency (milliseconds)', 'part22-latency2.png'
)
plot_something2(
    lambda x: x['total_latency'].std() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'latency standard deviation vs. threads per server', 'threads per server', 'latency standard deviation (milliseconds)', 'part22-latency_std2.png'
)
plot_something2(
    lambda x: (x['rtt_latency'] - x['inserver_latency']).mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Network latency vs. threads per server', 'threads per server', 'network latency (milliseconds)', 'part22-network_latency2.png'
)
plot_something2(
    lambda x: (x['rtt_latency'] - x['inserver_latency']).std() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Network latency standard deviation vs. threads per server', 'threads per server', 'network latency standard deviation (milliseconds)', 'part22-network_latency_std2.png'
)
plot_something2(
    lambda x: (x['inserver_latency'] - x['handle_latency']).mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Queue latency vs. threads per server', 'threads per server', 'queue latency (milliseconds)', 'part22-queue_latency2.png'
)

plt.cla()
plt.close("all")

#endregion

#region
#plot set 3: Fix batch_size x threads_per_server x hosts = 16
#            Compare performance vs. batch size, with different
#            number of hosts in the plot

def plot_something3(func, title, x_label, y_label, filename):
    # hosts = [1, 2, 4]
    # threads_per_server = [1, 2, 4, 8, 16]
    # batch_size = [1, 2, 4, 8, 16]
    data = [
        [ func(part22_stats[(x, 16//x, 1)]) for x in [1, 2, 4, 8, 16] ], 
        [ func(part22_stats[(x, 8//x, 2)]) for x in [1, 2, 4, 8] ],
        [ func(part22_stats[(x, 4//x, 4)]) for x in [1, 2, 4] ]
    ]

    fig, ax = plt.subplots()
    ax.plot([1, 2, 4, 8, 16], data[0], label='1 host')
    ax.plot([1, 2, 4, 8], data[1], label='2 hosts')
    ax.plot([1, 2, 4], data[2], label='4 hosts')

    ax.set_title(title)
    ax.set_xlabel(x_label)
    ax.set_ylabel(y_label)
    ax.legend()
    plt.savefig(os.path.join(fig_dir, filename))

plot_something3(
    lambda x: x['throughput'],
    'Throughput vs. batch size', 'batch size', 'throughput (images per second)', 'part22-throughput3.png'
)
plot_something3(
    lambda x: x['total_latency'].mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Latency vs. batch size', 'batch size', 'latency (milliseconds)', 'part22-latency3.png'
)
plot_something3(
    lambda x: x['total_latency'].std() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'latency standard deviation vs. batch size', 'batch size', 'latency standard deviation (milliseconds)', 'part22-latency_std3.png'
)
plot_something3(
    lambda x: (x['rtt_latency'] - x['inserver_latency']).mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Network latency vs. batch size', 'batch size', 'network latency (milliseconds)', 'part22-network_latency3.png'
)
plot_something3(
    lambda x: (x['rtt_latency'] - x['inserver_latency']).std() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Network latency standard deviation vs. batch size', 'batch size', 'network latency standard deviation (milliseconds)', 'part22-network_latency_std3.png'
)
plot_something3(
    lambda x: (x['inserver_latency'] - x['handle_latency']).mean() / (10**6), # convert to milliseconds (one millisecond is 10^6 nanoseconds)
    'Queue latency vs. batch size', 'batch size', 'queue latency (milliseconds)', 'part22-queue_latency3.png'
)


plt.cla()
plt.close("all")

#endregion

#region
#plot set 4: We see how servers will be chosen according to their latency.

b, t, h = 4, 4, 4
stat = part22_stats[(b, t, h)]
server_stat = {addr: [] for addr in set(stat['server_addr'])}
for idx, latency in enumerate(zip(
    stat['handle_latency'], stat['inserver_latency'],
    stat['rtt_latency'], stat['total_latency'], stat['ongoing_tasks'])):
    server_stat[stat['server_addr'][idx]].append(latency)

def plot_something4(func, title, x_label, y_label, filename):
    data = [ func(server_stat[addr]) for addr in server_stat ] 
    servers = [ addr.split(':')[0] for addr in server_stat ]

    fig, ax = plt.subplots()
    ax.bar(range(len(data)), data, tick_label=servers)

    ax.set_title(title)
    ax.set_xlabel(x_label)
    ax.set_ylabel(y_label)
    plt.savefig(os.path.join(fig_dir, filename))

func_throughput = lambda x: len(x) / stat['total_time'] * (10**9)
plot_something4(func_throughput, 'Throughput of each server', 'server', 'throughput (images per second)', 'part22-throughput4.png')

func_latency = lambda x: np.array(x)[:, 3].mean() / (10**6)
plot_something4(func_latency, 'Latency of each server', 'server', 'latency (milliseconds)', 'part22-latency4.png')

func_latency_std = lambda x: np.array(x)[:, 3].std() / (10**6)
plot_something4(func_latency_std, 'Latency standard deviation of each server', 'server', 'latency standard deviation (milliseconds)', 'part22-latency_std4.png')

func_network_latency = lambda x: (np.array(x)[:, 2] - np.array(x)[:, 1]).mean() / (10**6)
plot_something4(func_network_latency, 'Network latency of each server', 'server', 'network latency (milliseconds)', 'part22-network_latency4.png')

func_handle_latency = lambda x: np.array(x)[:, 0].mean() / (10**6)
plot_something4(func_handle_latency, 'Handle latency of each server', 'server', 'handle latency (milliseconds)', 'part22-handle_latency4.png')

plt.cla()
plt.close("all")

#endregion
