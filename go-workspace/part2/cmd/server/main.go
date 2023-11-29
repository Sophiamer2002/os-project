package main

import (
	"bytes"
	ctx "context"
	"flag"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"os-project/part12/pool"
	pb "os-project/part2/imgdownload"

	"github.com/nfnt/resize"
	"google.golang.org/grpc"
)

type imgDownloadServer struct {
	pb.UnimplementedImgDownloadServer

	pool          *pool.Pool
	ongoing_tasks int32
}

func (s *imgDownloadServer) GetSingleImg(ctx ctx.Context, in *pb.ImgRequest) (*pb.ImgResponse, error) {
	start := time.Now()
	atomic.AddInt32(&s.ongoing_tasks, 1)
	// In handler function: defer atomic.AddInt32(&s.ongoing_tasks, -1)

	c := make(chan *pb.ImgResponse)
	s.pool.AddTask(&pool.Task{
		Params:  []interface{}{in.Url, in.Sz.Width, in.Sz.Height, c, &s.ongoing_tasks},
		Handler: handler,
	})

	res := <-c
	res.InserverLatency = time.Since(start).Nanoseconds()
	return res, nil
}

func (s *imgDownloadServer) GetMultiImgs(stream pb.ImgDownload_GetMultiImgsServer) error {
	c := make(chan *pb.ImgResponse)
	var (
		send_err, recv_err error
		request            *pb.ImgRequest
		send, recv         int
		recv_all           bool
	)

	send, recv = 0, 0

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for obj := range c {
			obj.InserverLatency = int64(time.Now().Nanosecond()) - obj.InserverLatency // TODO: There is some problem here
			send_err = stream.Send(obj)
			if send_err != nil {
				log.Printf("Error sending response: %v", send_err)
				break
			}
			send += 1
			if recv_all && send == recv {
				break
			}
		}
	}()

	for {
		request, recv_err = stream.Recv()
		if recv_err != nil || send_err != nil {
			break
		}
		// In handler function: defer atomic.AddInt32(&s.ongoing_tasks, -1)
		atomic.AddInt32(&s.ongoing_tasks, 1)
		s.pool.AddTask(&pool.Task{
			Params:  []interface{}{request.Url, request.Sz.Width, request.Sz.Height, c, &s.ongoing_tasks, time.Now()},
			Handler: handler,
		})
		recv += 1
	}

	recv_all = true
	wg.Wait()

	if recv_err == io.EOF && send_err == nil {
		return nil
	} else if recv_err != nil {
		return recv_err
	} else {
		return send_err
	}
}

func handler(params ...interface{}) {
	start := time.Now()

	url := params[0].(string)
	width := uint(params[1].(uint32))
	height := uint(params[2].(uint32))
	c := params[3].(chan *pb.ImgResponse)
	ongoing := params[4].(*int32)
	var inserver_start time.Time
	if len(params) > 5 {
		inserver_start = params[5].(time.Time)
	}

	var (
		err error  = nil
		b   []byte = nil
	)

	defer atomic.AddInt32(ongoing, -1)
	defer func() {
		res := pb.ImgResponse{
			Img:             b,
			Url:             url,
			Success:         err == nil,
			ErrorMsg:        fmt.Sprintf("%v", err),
			OngoingRequests: atomic.LoadInt32(ongoing),
			HandleLatency:   time.Since(start).Nanoseconds(),
			InserverLatency: int64(inserver_start.Nanosecond()),
		}
		c <- &res
	}()

	// The following code deals with the image resizing
	// err: write any error to this variable
	// b: write the resized image to this variable
	// if any error occurs, return immediately

	// Step1: load the image
	if len(url) != 6 {
		err = fmt.Errorf("invalid url: %s", url)
		return
	}
	filename := filepath.Join("/ImageNet", url[0:2], url[2:4], url[4:6]+".JPEG")
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		return
	}

	// Step2: resize the image and return
	m := resize.Resize(width, height, img, resize.Lanczos3)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, m, nil)
	if err != nil {
		return
	}
	b = buf.Bytes()
}

func newServer(workers, capacity int) *imgDownloadServer {
	pool := pool.New(workers, capacity)
	pool.Run()
	log.Printf("Server started with %d workers and %d capacity.", workers, capacity)
	return &imgDownloadServer{pool: pool}
}

var (
	// Command line arguments
	threads  = flag.Int("n-t", 1, "Number of threads to use")
	capacity = flag.Int("cap", 100, "Capacity of the work queue")
	addr     = flag.String("addr", "localhost:51151", "IP address to listen on")

	// grpc
	opts []grpc.ServerOption
)

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterImgDownloadServer(grpcServer, newServer(*threads, *capacity))
	grpcServer.Serve(lis)
}
