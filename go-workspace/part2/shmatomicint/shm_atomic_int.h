#ifndef SHM_ATOMIC_INT_H_
#define SHM_ATOMIC_INT_H_

#include <fcntl.h>
#include <sys/mman.h>
#include <unistd.h>
#include <stdatomic.h>

void *shmint_new(char *name, int initial);
void *shmint_bind(char *name);
int shmint_unlink(char *name);

void shmint_atomic_store(void *ptr, int new_value);
int shmint_atomic_load(void *ptr);
int shmint_atomic_exchange(void *ptr, int new_value);
int shmint_atomic_compare_exchange(void *ptr, int expected, int new_value);
int shmint_atomic_fetch_add(void *ptr, int value);
int shmint_atomic_fetch_sub(void *ptr, int value);

#endif // SHM_ATOMIC_INT_H_
