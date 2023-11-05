package main

import (
	"flag"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"os-project/part12/pool"

	"github.com/nfnt/resize"
)

var (
	// Command line arguments
	cpus    = flag.Int("cpus", 1, "Number of CPUs to use")
	threads = flag.Int("n-t", 1, "Number of threads to use")
	cap     = flag.Int("cap", 100, "Capacity of the work queue")
	src_dir = flag.String("src-dir", "data", "source image directory")
	out_dir = flag.String("out-dir", "out", "output directory")
)

func resz(params ...interface{}) {
	if len(params) < 3 {
		panic("Too few params...")
	}

	src_dir, ok0 := params[0].(string)
	image_name, ok1 := params[1].(string)
	out_dir, ok2 := params[2].(string)

	if !ok0 || !ok1 || !ok2 {
		panic("Error interface types...")
	}

	filename := filepath.Join(src_dir, image_name)
	dest_filename := filepath.Join(out_dir, image_name)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Load picture failed", filename)
		panic(err)
	}

	img, err := jpeg.Decode(file)
	if err != nil {
		fmt.Println("Decode picture failed", filename)
		panic(err)
	}
	file.Close()

	m := resize.Resize(128, 128, img, resize.Lanczos3)

	out, err := os.Create(dest_filename)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	jpeg.Encode(out, m, nil)
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(*cpus)

	begin := time.Now()
	images, err := os.ReadDir(*src_dir)
	if err != nil {
		panic(err)
	}

	task_pool := pool.New(*threads, *cap)
	task_pool.Run()
	for _, image := range images {
		task_pool.AddTask(
			&pool.Task{
				Handler: resz,
				Params:  []interface{}{*src_dir, image.Name(), *out_dir},
			})
	}

	task_pool.Close()
	task_pool.Wait()

	fmt.Printf("%f\n", time.Since(begin).Seconds())
}
