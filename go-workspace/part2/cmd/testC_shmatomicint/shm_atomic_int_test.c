#include "../../shmatomicint/shm_atomic_int.h"
#include <stdio.h>

#include <sys/wait.h>

int main() {
    void *my_int = shmint_new("test", 0);
    if (my_int == (void *)-1) {
        printf("Error creating shared memory\n");
        return 1;
    }

    printf("ATOMIC+++Initial value: %d\n", shmint_atomic_load(my_int));

    // fork a new process and increment the value
    int pid = fork();
    for (int i=0; i<100000; i++) {
        shmint_atomic_fetch_add(my_int, 1);
    }

    printf("ATOMIC+++%s: %d\n", pid == 0 ? "Child" : "Parent", shmint_atomic_load(my_int));

    // wait for the child process to finish
    if (pid != 0) {
        wait(NULL);
        printf("ATOMIC+++Final value: %d\n", shmint_atomic_load(my_int));
        shmint_unlink("test");
    } else {
        return 0;
    }

    // compare with nonatomic int
    int handle = shm_open("test", O_RDWR | O_CREAT, 0600);
    if (handle == -1) {
        printf("Error creating shared memory\n");
        return 1;
    }

    if (ftruncate(handle, sizeof(int)) == -1) {
        printf("Error creating shared memory\n");
        return 1;
    }

    int *my_int2 = mmap((void *)0, sizeof(int), 
        PROT_READ | PROT_WRITE, MAP_SHARED, handle, 0);

    if (my_int2 == MAP_FAILED) {
        printf("Error creating shared memory\n");
        return 1;
    }

    *my_int2 = 0;
    printf("NONATO+++Initial value: %d\n", *my_int2);

    // fork a new process and increment the value
    pid = fork();
    for (int i=0; i<100000; i++) {
        (*my_int2)++;
    }

    printf("NONATO+++%s: %d\n", pid == 0 ? "Child" : "Parent", *my_int2);

    // wait for the child process to finish
    if (pid != 0) {
        wait(NULL);
        printf("NONATO+++Final value: %d\n", *my_int2);
        shmint_unlink("test");
    }
}