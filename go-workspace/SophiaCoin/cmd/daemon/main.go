package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os-project/SophiaCoin/pkg/crypto"
	"os-project/SophiaCoin/pkg/mempool"
	pri "os-project/SophiaCoin/pkg/primitives"
	"sync"

	pb "os-project/SophiaCoin/pkg/rpc"
	taskpool "os-project/part12/pool"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// stringSlice is a custom flag type, implements flag.Value interface
type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", []string(*s))
}
func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var (
	// Command line options
	peers      stringSlice
	ip         = flag.String("ip", "10.1.0.112", "IP to listen on")
	port       = flag.String("port", "51151", "Port to listen on")
	dir        = flag.String("dir", "/osdata/osgroup4/SophiaCoin", "SophiaCoin directory")
	difficulty = flag.Uint("difficulty", 4, "Difficulty of mining")

	// grpc
	clients = make(map[string]pb.BroadcastServiceClient)

	pool *mempool.Mempool
)

// Server definition
type Node struct {
	pb.UnimplementedBroadcastServiceServer
	pool     *mempool.Mempool
	taskPool *taskpool.Pool
}

func (n *Node) BroadcastTransaction(ctx context.Context, tx *pb.Transaction) (*empty.Empty, error) {
	the_tx, err := pri.Deserialize(tx.Transaction)
	if err != nil {
		return &empty.Empty{}, err
	}
	tx_, ok := the_tx.(*pri.Transaction)
	if !ok {
		return &empty.Empty{}, err
	}

	log.Printf("Received transaction %x\n", pri.Hash(tx_))

	// Verify transaction
	err = n.pool.AddTransaction(tx_)
	if err != nil {
		log.Printf("Failed to add transaction %x: %v\n", pri.Hash(tx_), err)
		log.Printf("Don't Panic: Maybe the transaction already exists.\n")
		return &empty.Empty{}, err
	}

	n.taskPool.AddTask(
		&taskpool.Task{
			Handler: func(params ...interface{}) {
				broadcastTransaction(tx)
			},
		},
	)
	return &empty.Empty{}, nil
}

func (n *Node) BroadcastBlock(stream pb.BroadcastService_BroadcastBlockServer) error {
	latestBlock, err := stream.Recv()
	if err != nil {
		log.Println(err)
		return err
	}

	block, err := pri.Deserialize(latestBlock.Block)
	latestBlock_, ok := block.(*pri.Block)
	if !ok || err != nil || latestBlock.HeaderOnly {
		return fmt.Errorf("invalid block")
	}

	log.Printf("Received block %d\n", latestBlock.BlockHeight)

	height, _ := n.pool.GetLatestInfo()
	if latestBlock.BlockHeight == height+1 {
		err = n.pool.AppendBlock(latestBlock_)
		if err != nil {
			log.Println(err)
			return err
		}

		n.taskPool.AddTask(
			&taskpool.Task{
				Handler: func(params ...interface{}) {
					broadcastBlock(latestBlock.BlockHeight, latestBlock_, n.pool)
				},
			},
		)

		return nil
	} else if latestBlock.BlockHeight <= height {
		return fmt.Errorf("block height %d is lower than current height %d", latestBlock.BlockHeight, height)
	}

	// The peer has a much higher block height, request the newest blocks
	// First find the common ancestor
	// Use binary backoff to find the common ancestor

	var backoff, curHeight uint32 = 1, height + 1
	var found bool = false
	for backoff > 0 {
		checkHeight := curHeight - backoff
		log.Printf("Requesting block header %d\n", checkHeight)
		err = stream.Send(&pb.BlockRequest{
			BlockHeight: checkHeight,
			HeaderOnly:  true,
		})

		if err != nil {
			log.Println(err)
			return err
		}

		response, err := stream.Recv()
		if err != nil {
			log.Println(err)
			return err
		}

		block, err := pri.Deserialize(response.Block)
		if err != nil {
			log.Println(err)
			return err
		}
		header, ok := block.(*pri.BlockHeader)
		if !ok {
			return fmt.Errorf("invalid block header")
		}

		myHash := n.pool.GetBlockHash(checkHeight)
		if myHash == pri.Hash(header) {
			found = true
		} else if checkHeight == 0 {
			return fmt.Errorf("failed to find common ancestor")
		} else {
			curHeight -= backoff
		}

		if found {
			backoff /= 2
		} else {
			backoff *= 2
		}
		for backoff >= curHeight {
			backoff /= 2
		}
	}

	blocks := []*pri.Block{}
	for i := curHeight; i <= latestBlock.BlockHeight; i++ {
		log.Printf("Requesting block %d\n", i)
		err = stream.Send(&pb.BlockRequest{
			BlockHeight: i,
			HeaderOnly:  false,
		})

		if err != nil {
			log.Println(err)
			return err
		}

		response, err := stream.Recv()
		if err != nil {
			log.Println(err)
			return err
		}

		block, err := pri.Deserialize(response.Block)
		if err != nil {
			log.Println(err)
			return err
		}
		the_block, ok := block.(*pri.Block)
		if !ok {
			return fmt.Errorf("invalid block")
		}

		blocks = append(blocks, the_block)
	}

	err = n.pool.SwitchChain(blocks, curHeight)
	if err == nil {
		height, last_block := n.pool.GetLatestInfo()
		n.taskPool.AddTask(
			&taskpool.Task{
				Handler: func(params ...interface{}) {
					broadcastBlock(height, last_block, n.pool)
				},
			},
		)
	}
	return err
}

func (n *Node) Handshake(ctx context.Context, addr *pb.Address) (*pb.Address, error) {
	log.Printf("Received handshake from %s:%s\n", addr.Ip, addr.Port)
	n.taskPool.AddTask(
		&taskpool.Task{
			Handler: func(params ...interface{}) {
				connect(net.JoinHostPort(addr.Ip, addr.Port))
			},
		},
	)
	return &pb.Address{
		Ip:   *ip,
		Port: *port,
	}, nil
}

func (n *Node) ConstructTransaction(ctx context.Context, tc *pb.TransactionConstruct) (*pb.Transaction, error) {
	sendPubkey, err := crypto.FromBytes(tc.SendAddr)
	if err != nil {
		return nil, err
	}

	receivePubkey, err := crypto.FromBytes(tc.RecvAddr)
	if err != nil {
		return nil, err
	}

	tx, err := n.pool.ConstructTransaction(sendPubkey, receivePubkey, tc.Amount, tc.Fee)
	if err != nil {
		return nil, err
	}

	txBytes, _ := pri.Serialize(tx)
	return &pb.Transaction{
		Transaction: txBytes,
	}, nil
}

func (n *Node) RequestTransactionsByPublicKey(
	request *pb.TransactionRequestByPublicKey,
	stream pb.BroadcastService_RequestTransactionsByPublicKeyServer) error {
	block := n.pool.GetBlock(request.BlockHeight)
	if block == nil {
		return fmt.Errorf("block not found")
	}
	counter := 0
	if pri.Hash(block) != pri.HashResult(request.BlockHash) {
		return fmt.Errorf("block hash mismatch")
	}
	pubkey, err := crypto.FromBytes(request.PublicKey)
	if err != nil {
		return err
	}
	for i, tx := range block.GetTransactions() {
		txBytes, err := pri.Serialize(&tx)
		if err != nil {
			return err
		}
		for _, idx := range tx.RelatesTo(*pubkey, true) {
			amount := n.pool.GetTxAmount(tx.GetTxIns()[idx])
			err = stream.Send(&pb.TransactionInfo{
				BlockHeight:      request.BlockHeight,
				BlockHash:        request.BlockHash,
				TransactionIndex: uint32(i),
				Transaction:      txBytes,
				InOutIdx:         uint32(idx),
				IsTxIn:           true,
				Amount:           amount,
				MerkleProof:      nil,
			})
			counter += 1
			if err != nil {
				return err
			}
		}

		for _, idx := range tx.RelatesTo(*pubkey, false) {
			amount := tx.GetTxOuts()[idx].GetValue()
			err = stream.Send(&pb.TransactionInfo{
				BlockHeight:      request.BlockHeight,
				BlockHash:        request.BlockHash,
				TransactionIndex: uint32(i),
				Transaction:      txBytes,
				InOutIdx:         uint32(idx),
				IsTxIn:           false,
				Amount:           amount,
				MerkleProof:      nil,
			})
			counter += 1
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func newNode(pool *mempool.Mempool) *Node {
	n := &Node{
		pool:     pool,
		taskPool: taskpool.New(4, 100),
	}
	n.taskPool.Run()
	return n
}

// Client definition
func broadcastTransaction(tx *pb.Transaction) {
	var wg sync.WaitGroup
	wg.Add(len(clients))
	for _, client := range clients {
		go func(client pb.BroadcastServiceClient) {
			defer wg.Done()
			client.BroadcastTransaction(context.Background(), tx)
		}(client)
	}
	wg.Wait()
}

func broadcastBlock(height uint32, block *pri.Block, pool *mempool.Mempool) {
	var wg sync.WaitGroup
	wg.Add(len(clients))
	log.Println(clients)
	for addr, client := range clients {
		go func(client pb.BroadcastServiceClient, addr string) {
			defer wg.Done()

			log.Printf("Send block %d to %s\n", height, addr)
			stream, err := client.BroadcastBlock(context.Background())
			if err != nil {
				log.Println(err)
				log.Printf("Failed to send block %d to %s\n", height, addr)
				return
			}
			defer stream.CloseSend()

			blockBytes, err := pri.Serialize(block)
			if err != nil {
				log.Println(err)
				return
			}

			msg := &pb.Block{
				Block:       blockBytes,
				BlockHeight: height,
				HeaderOnly:  false,
			}

			err = stream.Send(msg)
			if err != nil {
				log.Println(err)
				return
			}

			for {
				request, err := stream.Recv() // TODO, what if the peer is down?
				if err != nil {
					log.Println(err)
					break
				}

				if request.BlockHeight > height {
					log.Printf("Peer %s has a higher block height %d, aborting\n", addr, request.BlockHeight)
					break
				}

				the_block := pool.GetBlock(request.BlockHeight)
				if the_block == nil {
					log.Printf("Peer %s has a lower block height %d, aborting\n", addr, request.BlockHeight)
					break
				}

				var the_block_bytes []byte
				if request.HeaderOnly {
					the_block_bytes, err = pri.Serialize(the_block.GetHeader())
				} else {
					the_block_bytes, err = pri.Serialize(the_block)
				}

				if err != nil {
					log.Println(err)
					break
				}

				msg = &pb.Block{
					Block:       the_block_bytes,
					BlockHeight: request.BlockHeight,
					HeaderOnly:  request.HeaderOnly,
				}

				err = stream.Send(msg)
				if err != nil {
					log.Println(err)
					break
				}
			}

		}(client, addr)
	}
	wg.Wait()
}

func connect(addr string) {
	if _, ok := clients[addr]; ok {
		return
	}
	log.Printf("Connecting to %s\n", addr)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to %s: %v", addr, err)
		return
	}
	clients[addr] = pb.NewBroadcastServiceClient(conn)
	clients[addr].Handshake(context.Background(), &pb.Address{
		Ip:   *ip,
		Port: *port,
	})
}

func main() {
	// Parse command line options
	flag.Var(&peers, "peer", "Peer to connect to")
	flag.Parse()

	pool := mempool.NewMempool(*dir, (uint32)(*difficulty))

	addr := net.JoinHostPort(*ip, *port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterBroadcastServiceServer(grpcServer, newNode(pool))
	go grpcServer.Serve(lis)

	for _, host := range peers {
		go connect(host)
	}

	for {
		height, _ := pool.GetLatestInfo()
		log.Printf("Mining a block %d...\n", height+1)
		pool.Mine()
		height, _ = pool.GetLatestInfo()
		log.Printf("Mined a block, trying to append it to the chain at height %d.\n", height)
		err := pool.AppendBlock(nil)
		if err != nil {
			log.Println(err)
			continue
		}
		height, block := pool.GetLatestInfo()
		go broadcastBlock(height, block, pool)
	}
}
