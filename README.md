# Operating System and Ditributed System -- Course Project

Yiying Wang, Yao Class 02, 2020011604  
Chengda Lu, Yao Class 12, 2021010899  
Fangyan Shi, Yao Class 12, 2021010892

## Project 1: Distributed Image Server

### Part 1 The Basics

#### Part 1.1 A fixed-sized bounded queue

Just run `make part11` in the root directory, and you can view `part11.pdf` in the root directory for our results.

For further information, you can see:
+ The source code is in `./go-workspace/part11`, in which we implement a channel-based queue in a package called **queue**, lying in `./go-workspace/part11/queue`.
+ `./temp/data/cap_{}_threads_{}.txt`, where the two parameters indicate the capacity of the queue and the number of threads(actually goroutines), respectively. The file includes the data that we use to draw the CDF and throughput curve. Each file consists of three lines:
    - The first line has two numbers `m` and `n`, which are the number of enqueue, dequeue operations(within 10 milliseconds), respectively
    - The second line has `m` integers, indicating the waiting time of each enqueue operation, in nanosecond.
    - The third line has `n` integers, indicating the waiting time of each dequeue operation, in nanosecond.

#### Part 1.2 Resizing 10k images concurrently

Just run 
```sh
make part12 IMAGE_NET_TEST_DIR=/path/to/imagnet/test/split IMG_DIR=/path/to/output/image
``` 
in the root directory, and you can view `part12.pdf` in the root directory for our results. The default value for these two variables are `/tiny-imagenet-200/test/images` and `/osdata/osgroup4/generated_imgs`. Change them in case they are unavailable.

For further information,   
+ The source code is in `./go-workspace/part12`, where the `os-project/part12` module lies. We implement a package `pool` based on the `queue` we implemented in Part 1.1, which can be used as a thread pool.
+ `./temp/data/part12.csv`: This file record the experiment result in a table, just as following   

    | cpus |capacity|threads|time(s)|
    | :----: | :----: | :----: | :----: |
    |  16   |   100 |  8  | 4.5  |

    The example table indicates that we use 4.5s to resize all 10000 pictures with 16 cpus, 8 worker goroutines, 100 buffer of the queue.

### Part 2 Scale the system!

<!-- TODO -->
