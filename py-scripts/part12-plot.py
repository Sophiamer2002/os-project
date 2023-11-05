import os
import argparse

import csv
from matplotlib import pyplot as plt

parser = argparse.ArgumentParser()
parser.add_argument('--data-file', type=str, help='The data file(.csv)')
parser.add_argument('--out-dir', type=str, help='Directory to save the plots')

args = parser.parse_args()

data = csv.reader(open(args.data_file, 'r'), delimiter=',')

# The first plot

fig, ax = plt.subplots(figsize=(20, 10))
ax.set_ylabel('Work Time(s)')
ax.set_xlabel('Number of Threads')

# read the first line
line = next(data)


