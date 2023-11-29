package main

import (
	"flag"
	"fmt"
	"os-project/part2/shmatomicint"
)

var (
	// Command line arguments
	isMaster = flag.Bool("master", false, "Is this the master instance?")
)

func main() {
	flag.Parse()
	var err error
	var shm *shmatomicint.ShmAtomicInt
	if *isMaster {
		// Master instance
		shm, err = shmatomicint.New("test", 0)
	} else {
		shm, err = shmatomicint.Bind("test")
	}

	if err != nil {
		panic(err)
	}

	shm.AtomicFetchAdd(1)

	fmt.Println("Waiting for other instances to finish...")
	for shm.AtomicLoad() < 2 {
	}

	for i := 0; i < 999999; i++ {
		shm.AtomicFetchAdd(1)
	}

	fmt.Printf("Final value: %d\n", shm.AtomicLoad())

	if *isMaster {
		err = shm.Unlink()
		if err != nil {
			panic(err)
		}
	}
}
