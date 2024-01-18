package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"os-project/SophiaCoin/cmd/client/cli"
	pri "os-project/SophiaCoin/pkg/primitives"
	pb "os-project/SophiaCoin/pkg/rpc"
	"os-project/SophiaCoin/pkg/wallet"
	taskPool "os-project/part12/pool"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// Command line options
	daemon = flag.String("daemon", "10.1.0.112:51151", "Daemon to connect to")
	ip     = flag.String("ip", "127.0.0.1", "IP to listen on")
	port   = flag.String("port", "51152", "Port to listen on")
	dir    = flag.String("dir", "/osdata/osgroup4/SophiaCoin", "SophiaCoin directory")

	server pb.BroadcastServiceClient
)

type Client struct {
	pb.UnimplementedBroadcastServiceServer

	wallet   *wallet.Wallet
	taskPool *taskPool.Pool
}

func newClient(dir string) *Client {
	// TODO
	c := &Client{
		wallet:   wallet.NewWallet(dir),
		taskPool: taskPool.New(2, 100),
	}

	c.taskPool.Run()

	return c
}

func (c *Client) BroadcastTransaction(ctx context.Context, tx *pb.Transaction) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (c *Client) BroadcastBlock(stream pb.BroadcastService_BroadcastBlockServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	block, err := pri.Deserialize(msg.Block)
	if err != nil {
		return err
	}

	latestBlock, ok := block.(*pri.Block)
	if !ok {
		return err
	}

	// log.Printf("Received block %v\n", latestBlock.GetHeader())

	height, _ := c.wallet.GetLatestInfo()
	if height >= msg.BlockHeight {
		return nil
	}

	headers := []*pri.BlockHeader{latestBlock.GetHeader()}
	for i := msg.BlockHeight - 1; ; i-- {
		if headers[len(headers)-1].VerifyPreviousHash(c.wallet.GetHeaderHash(i)) {
			break
		}
		if i == 0 {
			return nil
		}
		stream.Send(&pb.BlockRequest{
			BlockHeight: i,
			HeaderOnly:  true,
		})
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		blockHeader, err := pri.Deserialize(msg.Block)
		if err != nil {
			return err
		}

		header, ok := blockHeader.(*pri.BlockHeader)
		if !ok {
			return err
		}

		headers = append(headers, header)
	}

	// reverse headers
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-i-1] = headers[len(headers)-i-1], headers[i]
	}

	err = c.wallet.UpdateHeaders(msg.BlockHeight+1-uint32(len(headers)), headers...)
	if err != nil {
		return nil
	}

	height, _ = c.wallet.GetLatestInfo()

	keys := c.wallet.GetSelfAddress()
	for i := msg.BlockHeight + 1 - uint32(len(headers)); i <= height; i++ {
		for name, key := range keys {
			if err != nil {
				panic(err)
			}

			c.taskPool.AddTask(&taskPool.Task{
				Handler: func(params ...interface{}) {
					i := params[0].(uint32)
					name := params[1].(string)
					key := params[2].([]byte)
					var err error = fmt.Errorf("dummy")
					var stream pb.BroadcastService_RequestTransactionsByPublicKeyClient
					blockHash := c.wallet.GetHeaderHash(i)
					// log.Printf("Block %d: Fetching records...\n", i)
					for err != nil {
						stream, err = server.RequestTransactionsByPublicKey(
							context.Background(),
							&pb.TransactionRequestByPublicKey{
								BlockHeight: i,
								PublicKey:   key,
								BlockHash:   blockHash[:],
							},
						)
					}

					records := []*wallet.TxRecord{}

					var res *pb.TransactionInfo
					for res, err = stream.Recv(); err == nil; res, err = stream.Recv() {
						tx, err := pri.Deserialize(res.Transaction)
						if err != nil {
							continue
						}
						tx_, ok := tx.(*pri.Transaction)
						if !ok {
							continue
						}

						res_block_hash := pri.HashResult(res.BlockHash)
						record := wallet.NewRecord(
							int(res.BlockHeight),
							res_block_hash,
							pri.Hash(tx_),
							int(res.TransactionIndex),
							res.IsTxIn,
							int(res.InOutIdx),
							int(res.Amount),
							name,
							res.MerkleProof,
						)

						records = append(records, &record)
					}

					// if err != nil {
					// 	log.Printf("Block %d: Fetching records failed: %v\n", i, err)
					// }

					// log.Printf("Block %d: Fetching %d records...\n", i, len(records))

					c.wallet.AddTxRecords(records...)
				},
				Params: []interface{}{i, name, key},
			})
		}
	}

	return nil
}

func (c *Client) ConstructTransaction(ctx context.Context, request *pb.TransactionConstruct) (*pb.Transaction, error) {
	return nil, nil
}

func (c *Client) Handshake(ctx context.Context, addr *pb.Address) (*pb.Address, error) {
	log.Printf("Received handshake from %s:%s\n", addr.Ip, addr.Port)
	return &pb.Address{
		Ip:   *ip,
		Port: *port,
	}, nil
}

func (c *Client) RequestTransactionsByPublicKey(
	request *pb.TransactionRequestByPublicKey,
	stream pb.BroadcastService_RequestTransactionsByPublicKeyServer) error {
	return nil
}

func connect(addr string) error {
	log.Printf("Connecting to %s\n", addr)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to %s: %v", addr, err)
		return err
	}
	server = pb.NewBroadcastServiceClient(conn)
	_, err = server.Handshake(context.Background(), &pb.Address{
		Ip:   *ip,
		Port: *port,
	})
	return err
}

func main() {
	flag.Parse()

	addr := net.JoinHostPort(*ip, *port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}
	grpcServer := grpc.NewServer()
	client := newClient(*dir)
	pb.RegisterBroadcastServiceServer(grpcServer, client)
	go grpcServer.Serve(lis)

	err = connect(*daemon)

	if err != nil {
		log.Printf("Failed to connect to daemon, Exiting...")
		return
	}

	// recv_addr, _ := hex.DecodeString("3059301306072a8648ce3d020106082a8648ce3d03010703420004993be5b8329fdbf4a23a43bc7406a0de33fd8708f2676e9ec04ef97e78a34db4c6a5a82102897508ddf687ad8eaf3bd298616971d18c4c5b9a2f4ebe821439b3")
	// res, err := server.ConstructTransaction(context.Background(), &pb.TransactionConstruct{
	// 	SendAddr: client.wallet.GetAddress()["miner"],
	// 	RecvAddr: recv_addr,
	// 	Amount:   4053,
	// 	Fee:      43,
	// })

	// if err != nil {
	// 	panic(err)
	// }

	// tx, _ := pri.Deserialize(res.Transaction)
	// tx_ := tx.(*pri.Transaction)
	// err = client.wallet.SignTransaction(tx_, "miner")
	// if err != nil {
	// 	panic(err)
	// }
	// tx_bytes, _ := pri.Serialize(tx_)
	// server.BroadcastTransaction(context.Background(), &pb.Transaction{
	// 	Transaction: tx_bytes,
	// })

	// time.Sleep(time.Second * 100)

	// blockHash := client.wallet.GetHeaderHash(6)
	// server.RequestTransactionsByPublicKey(context.Background(), &pb.TransactionRequestByPublicKey{
	// 	BlockHeight: 6,
	// 	PublicKey:   client.wallet.GetAddress()["miner"],
	// 	BlockHash:   blockHash[:],
	// })

	cli.NewCli(&server, client.wallet).Start()
}
