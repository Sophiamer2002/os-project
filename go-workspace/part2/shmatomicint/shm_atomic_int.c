#include "shm_atomic_int.h"
#include <sys/shm.h>

void *shmint_new(char *name, int initial) {
    int handle = shm_open(name, O_CREAT | O_EXCL | O_RDWR, 0600);
    if (handle == -1) {
        return (void *)-1;
    }

    if (ftruncate(handle, sizeof(atomic_int)) == -1) {
        return (void *)-1;
    }

    atomic_int *ptr = mmap((void *)0, sizeof(atomic_int), 
        PROT_READ | PROT_WRITE, MAP_SHARED, handle, 0);

    if (ptr == MAP_FAILED) {
        shm_unlink(name);
        return (void *)-1;
    }

    atomic_init(ptr, initial);
    return ptr;
}

void *shmint_bind(char *name) {
    int handle = shm_open(name, O_RDWR, 0);
    if (handle == -1) {
        return (void *)-1;
    }

    atomic_int *ptr = mmap((void *)0, sizeof(atomic_int), 
        PROT_READ | PROT_WRITE, MAP_SHARED, handle, 0);
    
    if (ptr == MAP_FAILED) {
        return (void *)-1;
    }

    return ptr;
}

int shmint_unlink(char *name) {
    return shm_unlink(name);
}

void shmint_atomic_store(void *ptr, int new_value) {
    atomic_store((atomic_int *)ptr, new_value);
}

int shmint_atomic_load(void *ptr) {
    return atomic_load((atomic_int *)ptr);
}

int shmint_atomic_exchange(void *ptr, int new_value) {
    return atomic_exchange((atomic_int *)ptr, new_value);
}

int shmint_atomic_compare_exchange(void *ptr, int expected, int new_value) {
    return atomic_compare_exchange_strong((atomic_int *)ptr, &expected, new_value);
}

int shmint_atomic_fetch_add(void *ptr, int value) {
    return atomic_fetch_add((atomic_int *)ptr, value);
}

int shmint_atomic_fetch_sub(void *ptr, int value) {
    return atomic_fetch_sub((atomic_int *)ptr, value);
}
