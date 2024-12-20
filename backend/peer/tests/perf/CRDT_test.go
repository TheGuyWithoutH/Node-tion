//go:build performance
// +build performance

package perf

import (
	z "Node-tion/backend/internal/testing"
	"Node-tion/backend/peer/tests"
	"Node-tion/backend/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
	"os"
	"testing"
	"time"
)

//-----------------------------------------------------------------------------------------------
// Run Benchmark: 1 node, 1 round, 10 operations
// Time taken for initial document loading and rendering in the editor.

// This test executes the exact same function as the BenchmarkCRDT below.
// Its goal is mainly to raise any error that could occur during its execution as the benchmark hides them.
func Test_CRDT_Benchmark_Correctness(t *testing.T) {
	runCRDT(t, 1, 1, 10)
}

// Run BenchmarkCRDT and compare results to reference assessments.
// TODO Run as follow: make test_bench_crdt
func Test_CRDT_BenchmarkCRDT(t *testing.T) {
	// run the benchmark
	res := testing.Benchmark(BenchmarkCRDT)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 10_000, 500_000}, // TODO 4906, 315952
		{"allocs ok", 20_000, 1_000_000},
		{"allocs passable", 50_000, 2_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 500 * time.Millisecond}, // TODO 217.162025ms
		{"speed ok", 1 * time.Second},
		{"speed passable", 2 * time.Second},
	})
}

// Spam a node with CRDT messages and check that it correctly processed them. We
// are going to randomly send messages filling the interval [0,N), and [N+1,2N].
// In this case the node should process the [0, N) CRDT messages, update its Editor
// and compile the document.
func BenchmarkCRDT(b *testing.B) {

	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDT(b, 1, b.N, 10)
}

//-----------------------------------------------------------------------------------------------
// Run Benchmark: 100 operations, 1 round, variable number of nodes (2, 5, 10)
// Storage / bandwidth overhead of the CRDT.

// This test executes the exact same function as the BenchmarkCRDTMultiNodes below.
// Its goal is mainly to raise any error that could occur during its execution as the benchmark hides them.
func Test_CRDT_Benchmark_Correctness_Multi_Nodes(t *testing.T) {
	runCRDT(t, 2, 1, 100)
	runCRDT(t, 5, 1, 100)
	runCRDT(t, 10, 1, 100)
}

// Run runCRDT and compare results to reference assessments.
func Test_CRDT_BenchmarkCRDTMultiNodes(t *testing.T) {
	// run the benchmark for 2 nodes
	res := testing.Benchmark(BenchmarkCRDTTwoNodes)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 200_000, 20_000_000}, // TODO 137827, 10761842
		{"allocs ok", 400_000, 40_000_000},
		{"allocs passable", 750_000, 75_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 500 * time.Millisecond}, // TODO 357.375486ms
		{"speed ok", 1 * time.Second},
		{"speed passable", 2 * time.Second},
	})

	// run the benchmark for 5 nodes
	res = testing.Benchmark(BenchmarkCRDTFiveNodes)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 300_000, 30_000_000}, // TODO 218492, 19107136
		{"allocs ok", 500_000, 50_000_000},
		{"allocs passable", 1_000_000, 100_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 1 * time.Second}, // TODO 744.088312ms
		{"speed ok", 2 * time.Second},
		{"speed passable", 5 * time.Second},
	})

	// run the benchmark for 5 nodes
	res = testing.Benchmark(BenchmarkCRDTTenNodes)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 500_000, 50_000_000}, // TODO 413671, 33072456
		{"allocs ok", 1_000_000, 100_000_000},
		{"allocs passable", 1_500_000, 150_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 2 * time.Second}, // TODO 1.38834375s
		{"speed ok", 5 * time.Second},
		{"speed passable", 10 * time.Second},
	})
}

func BenchmarkCRDTTwoNodes(b *testing.B) {

	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDT(b, 2, b.N, 100)
}

// Benchmark for 5 nodes
func BenchmarkCRDTFiveNodes(b *testing.B) {
	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDT(b, 5, b.N, 100)
	// TODO results
	// with 100 ops - ~2s
	// with 200 ops - ~23s
	// with 500 ops - >1m43s
}

// Benchmark for 10 nodes
func BenchmarkCRDTTenNodes(b *testing.B) {
	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDT(b, 10, b.N, 100)
}

//-----------------------------------------------------------------------------------------------
// Run Benchmark: 1 node, 1 round, variable number of operations (100, 1000, 10000)
// Performance characteristics at scale, with >1000s of operations.

func Test_CRDT_Benchmark_Correctness_Scale_Ops(t *testing.T) {
	runCRDT(t, 1, 1, 100)
	runCRDT(t, 1, 1, 1000)
	runCRDT(t, 1, 1, 10000)
}

// Run runCRDT and compare results to reference assessments.
func Test_CRDT_BenchmarkCRDT_Scale_Ops(t *testing.T) {
	// run the benchmark for 2 nodes
	res := testing.Benchmark(BenchmarkCRDTSmallOps)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 100_000, 5_000_000}, // TODO 37641, 2456243
		{"allocs ok", 200_000, 10_000_000},
		{"allocs passable", 500_000, 20_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 500 * time.Millisecond}, // TODO 233.449358ms
		{"speed ok", 1 * time.Second},
		{"speed passable", 2 * time.Second},
	})

	// run the benchmark for 5 nodes
	res = testing.Benchmark(BenchmarkCRDTMediumOps)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 500_000, 50_000_000}, // TODO 389084, 29604064
		{"allocs ok", 750_000, 75_000_000},
		{"allocs passable", 1_000_000, 100_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 1 * time.Second}, // TODO 527.740027ms
		{"speed ok", 2 * time.Second},
		{"speed passable", 5 * time.Second},
	})

	// run the benchmark for 5 nodes
	res = testing.Benchmark(BenchmarkCRDTLargeOps)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 5_000_000, 1_000_000_000}, // TODO 4649973, 846101568
		{"allocs ok", 10_000_000, 5_000_000_000},
		{"allocs passable", 15_000_000, 10_000_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 10 * time.Second}, // TODO 6.731827583s
		{"speed ok", 15 * time.Second},
		{"speed passable", 20 * time.Second},
	})
}

// Benchmark for 100 ops
func BenchmarkCRDTSmallOps(b *testing.B) {

	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDT(b, 1, b.N, 100)
}

// Benchmark for 1000 ops
func BenchmarkCRDTMediumOps(b *testing.B) {
	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDT(b, 1, b.N, 1000)
}

// Benchmark for 10000 ops
func BenchmarkCRDTLargeOps(b *testing.B) {
	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDT(b, 1, b.N, 10000)
}

func runCRDT(t require.TestingT, nodeCount, rounds, opN int) {
	// run as many times as specified by rounds
	for i := 0; i < rounds; i++ {
		rand.Seed(1)

		transp := channelFac()
		nodes := make([]z.TestNode, nodeCount)

		// Instantiate nodes
		for i := 0; i < nodeCount; i++ {
			nodes[i] = z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
		}

		// Add peers => fully connected network
		for i := 0; i < nodeCount; i++ {
			for j := 0; j < nodeCount; j++ {
				if i != j {
					nodes[i].AddPeer(nodes[j].GetAddr())
				}
			}
		}

		// Create two different blocks in the first node
		docID := "0@" + nodes[0].GetAddr()
		blockID1 := "1@" + nodes[0].GetAddr()
		blockID2 := "2@" + nodes[0].GetAddr()

		op1 := tests.CreateNewBlockOp(nodes[0].GetAddr(), docID, blockID1)
		op2 := tests.CreateNewBlockOp(nodes[0].GetAddr(), docID, blockID2)

		crdtMsg1 := types.CRDTOperationsMessage{Operations: op1}
		crdtMsg2 := types.CRDTOperationsMessage{Operations: op2}

		err := nodes[0].SaveTransactions(crdtMsg1)
		require.NoError(t, err)
		err = nodes[0].SaveTransactions(crdtMsg2)
		require.NoError(t, err)

		time.Sleep(time.Millisecond * 100)

		// Generate random operations for both blocks and send them randomly between nodes
		ops := make([]types.CRDTOperation, opN)
		for i := 0; i < opN; i++ {
			char := string(rune(rand.Intn(26) + 97))
			blockID := blockID1
			if rand.Intn(2) == 0 {
				blockID = blockID2
			}
			nodeIndex := rand.Intn(nodeCount)
			ops[i] = tests.CreateInsertsFromString(char, nodes[nodeIndex].GetAddr(), docID, blockID, i+1)[0]
		}

		// Send operations
		for _, op := range ops {
			nodeIndex := rand.Intn(nodeCount)
			crdtMsg := types.CRDTOperationsMessage{
				Operations: []types.CRDTOperation{op},
			}
			err := nodes[nodeIndex].SaveTransactions(crdtMsg)
			require.NoError(t, err)
		}

		// Wait proportionally to the number of messages
		for i := 0; i < nodeCount*100; i++ {
			time.Sleep(time.Millisecond)
		}

		// Compile document in each node to ensure consistency
		for _, node := range nodes {
			_, err := node.CompileDocument(docID)
			require.NoError(t, err)
		}

		// Cleanup
		for _, node := range nodes {
			node.Stop()
		}
	}
}
