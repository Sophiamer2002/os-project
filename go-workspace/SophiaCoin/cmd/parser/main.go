package main

import (
	"flag"
	"fmt"
	"os"
	pri "os-project/SophiaCoin/pkg/primitives"
)

var (
	// command line arguments
	the_file = flag.String("file", "/osdata/osgroup4/SophiaCoin/blocks/Block6.dat", "The file to parse")
	out_file = flag.String("out", "", "The file to output to")
)

func main() {
	flag.Parse()
	b, err := os.ReadFile(*the_file)
	if err != nil {
		panic(err)
	}

	data, err := pri.Deserialize(b)
	if err != nil {
		panic(err)
	}

	s := fmt.Sprintf("%v", data)
	os.WriteFile(*out_file, []byte(s), 0644)

	fmt.Println(s)
}
