import os
import csv
import logging

logging.basicConfig(format='%(asctime)s %(levelname)s:%(message)s', level=logging.INFO)

import numpy as np
import matplotlib.pyplot as plt

top_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
tmp_dir = bin_dir = os.path.join(top_dir, 'temp')
fig_dir = os.path.join(tmp_dir, 'fig')
data_dir = os.path.join(tmp_dir, 'data')
 
def read_stats(path: str) -> dict:
    with open(path, 'r') as f:
        reader = csv.reader(f)
        transposed = list(zip(*reader))
        return {
            t[0]: nparray(t[1:])
            for t in transposed
        }

def get_total_time(path: str) -> float:
    with open(path, 'r') as f:
        times = [float(x.strip()) for x in f.readlines()]
        return sum(times) / len(times)

def nparray(data: list) -> np.ndarray:
    try:
        return np.array(data, dtype=np.float64)
    except ValueError:
        return np.array(data)

def get_stats(server_thread: int, client_thread: int) -> dict:
    stats_file = os.path.join(data_dir, f'stats{client_thread}_serverthreads{server_thread}.csv')
    time_file = os.path.join(data_dir, f'time{client_thread}_serverthreads{server_thread}.txt')
    a = read_stats(stats_file)
    a['total_time'] = get_total_time(time_file)
    a['throughput'] = 2000 * (10**9) / a['total_time']  # time is measured in nanoseconds
    a['server_thread'] = server_thread
    a['client_thread'] = client_thread
    return a

#region The first set of plots: performance vs. number of client threads

server_threads = [1, 2, 4, 8, 12, 24]
client_threads = [1, 2, 4, 8, 12, 16]

stats = {
    (server_thread, client_thread): get_stats(server_thread, client_thread)
    for server_thread in server_threads
    for client_thread in client_threads
}

def plot_something(func, title, x_label, y_label, filename):
    data = [
        [
            func(stats[(server_thread, client_thread)])
            for server_thread in server_threads
        ] for client_thread in client_threads
    ]

    fig, ax = plt.subplots()
    for i in range(len(client_threads)):
        ax.plot(server_threads, data[i], label=f'{client_threads[i]} client threads')
    
    ax.set_title(title)
    ax.set_xlabel(x_label)
    ax.set_ylabel(y_label)
    ax.legend()
    fig.savefig(filename)

throughput = lambda s: s['throughput']
plot_something(throughput, 'Throughput vs. number of server threads', 'Number of server threads', 'Throughput (requests per second)', os.path.join(fig_dir, 'part21-throughput.png'))

speed_up = lambda s: throughput(s) / throughput(stats[(1, 1)])
plot_something(speed_up, 'Speed up vs. number of server threads', 'Number of server threads', 'Speed up', os.path.join(fig_dir, 'part21-speed_up.png'))

latency = lambda s: s['total_latency'].mean() / (10**6) # convert to milliseconds (one millisecond is 10^6 nanoseconds)
plot_something(latency, 'Latency vs. number of server threads', 'Number of server threads', 'Latency (milliseconds)', os.path.join(fig_dir, 'part21-latency.png'))

latency_std = lambda s: s['total_latency'].std() / (10**6)
plot_something(latency_std, 'Latency standard deviation vs. number of server threads', 'Number of server threads', 'Latency standard deviation (milliseconds)', os.path.join(fig_dir, 'part21-latency_std.png'))

network_latency = lambda s: (s['rtt_latency'] - s['inserver_latency']).mean() / (10**6)
plot_something(network_latency, 'Network latency vs. number of server threads', 'Number of server threads', 'Network latency (milliseconds)', os.path.join(fig_dir, 'part21-network_latency.png'))

queue_latency = lambda s: (s['inserver_latency'] - s['handle_latency']).mean() / (10**6)
plot_something(queue_latency, 'Queue latency vs. number of server threads', 'Number of server threads', 'Queue latency (milliseconds)', os.path.join(fig_dir, 'part21-queue_latency.png'))

queue_length = lambda s: s['ongoing_tasks'].mean()
plot_something(queue_length, 'Queue length vs. number of server threads', 'Number of server threads', 'Queue length', os.path.join(fig_dir, 'part21-queue_length.png'))

#endregion

#region The second set of plots: performance vs. number of client processes

def get_stats2(client_nums, client_threads):
    stats_file = os.path.join(data_dir, f'stats{client_nums}x{client_threads}.csv')
    time_file = os.path.join(data_dir, f'time{client_nums}x{client_threads}.txt')
    a = read_stats(stats_file)
    a['total_time'] = get_total_time(time_file)
    a['throughput'] = 2000 * (10**9) / a['total_time']  # time is measured in nanoseconds
    a['client_nums'] = client_nums
    a['client_threads'] = client_threads
    return a

total_threads = 8
client_nums = [1, 2, 4, 8]
stats2 = {
    (client_nums, total_threads // client_nums): get_stats2(client_nums, total_threads // client_nums)
    for client_nums in client_nums
}

def plot_something2(func, title, y_label, filename):
    data = [
        func(stats2[(client_nums, total_threads // client_nums)])
        for client_nums in client_nums
    ]

    fig, ax = plt.subplots()
    ax.plot(client_nums, data)
    
    ax.set_title(title)
    ax.set_xlabel("Number of client processes")
    ax.set_ylabel(y_label)
    fig.savefig(filename)

throughput2 = lambda s: s['throughput']
plot_something2(throughput2, 'Throughput vs. number of client processes', 'Throughput (requests per second)', os.path.join(fig_dir, 'part21-throughput2.png'))

latency2 = lambda s: s['total_latency'].mean() / (10**6) # convert to milliseconds (one millisecond is 10^6 nanoseconds)
plot_something2(latency2, 'Latency vs. number of client processes', 'Latency (milliseconds)', os.path.join(fig_dir, 'part21-latency2.png'))

latency_std2 = lambda s: s['total_latency'].std() / (10**6)
plot_something2(latency_std2, 'Latency standard deviation vs. number of client processes', 'Latency standard deviation (milliseconds)', os.path.join(fig_dir, 'part21-latency_std2.png'))

network_latency2 = lambda s: (s['rtt_latency'] - s['inserver_latency']).mean() / (10**6)
plot_something2(network_latency2, 'Network latency vs. number of client processes', 'Network latency (milliseconds)', os.path.join(fig_dir, 'part21-network_latency2.png'))

queue_latency2 = lambda s: (s['inserver_latency'] - s['handle_latency']).mean() / (10**6)
plot_something2(queue_latency2, 'Queue latency vs. number of client processes', 'Queue latency (milliseconds)', os.path.join(fig_dir, 'part21-queue_latency2.png'))

queue_length2 = lambda s: s['ongoing_tasks'].mean()
plot_something2(queue_length2, 'Queue length vs. number of client processes', 'Queue length', os.path.join(fig_dir, 'part21-queue_length2.png'))

#endregion
