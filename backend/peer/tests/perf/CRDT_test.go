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

// This test executes the exact same function as the BenchmarkCRDTSingle below.
// Its goal is mainly to raise any error that could occur during its execution as the benchmark hides them.
func Test_CRDT_Single_Doc_Benchmark_Correctness(t *testing.T) {
	runCRDTSingle(t, 10)
}

// Run BenchmarkCRDTSingle and compare results to reference assessments
func Test_CRDT_Single_Doc_BenchmarkCRDTSingle(t *testing.T) {
	// run the benchmark
	res := testing.Benchmark(BenchmarkCRDTSingle)
	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 100 * time.Millisecond},
		{"speed ok", 1 * time.Second},
		{"speed passable", 5 * time.Second},
	})
	res = testing.Benchmark(BenchmarkCRDTSingle)
}

// Calculate the time it takes to process N CRDT operations and
// compile the document. The operations are randomly generated.
func BenchmarkCRDTSingle(b *testing.B) {
	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()
	runCRDTSingle(b, 10)
}
func runCRDTSingle(t require.TestingT, opN int) {
	transp := channelFac()
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	docID := "0@" + node1.GetAddr()
	blockID := "1@" + node1.GetAddr()
	op := tests.CreateNewBlockOp(node1.GetAddr(), docID, blockID)
	crdtMgs := types.CRDTOperationsMessage{
		Operations: op,
	}
	err := node1.SaveTransactions(crdtMgs)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	// Generate opN random operations
	ops := make([]types.CRDTOperation, opN)

	for i := 0; i < opN; i++ {
		// random character
		char := string(rune(rand.Intn(26) + 97))
		ops[i] = tests.CreateInsertsFromString(char, node1.GetAddr(), docID, blockID, i)[0]
	}
	for _, op := range ops {
		crdtMsg := types.CRDTOperationsMessage{
			Operations: []types.CRDTOperation{op},
		}
		err := node1.SaveTransactions(crdtMsg)
		require.NoError(t, err)
	}
	_, err = node1.CompileDocument(docID)
	require.NoError(t, err)
	// cleanup
	node1.Stop()
}

//-----------------------------------------------------------------------------------------------
// Run Benchmark: 100 operations, 1 round, variable number of nodes (2, 5, 10)
// Storage / bandwidth overhead of the CRDT.

// This test executes the exact same function as the BenchmarkCRDTMultiNodes below.
// Its goal is mainly to raise any error that could occur during its execution as the benchmark hides them.
func Test_CRDT_Benchmark_Correctness_Multi_Nodes(t *testing.T) {
	runCRDT(t, 1, 1, 100)
	runCRDT(t, 2, 1, 100)
	runCRDT(t, 5, 1, 100)
	runCRDT(t, 10, 1, 100)
}

// Run runCRDT and compare results to reference assessments.
func Test_CRDT_BenchmarkCRDTMultiNodes(t *testing.T) {
	// run the benchmark for 1 node
	res := testing.Benchmark(BenchmarkCRDT)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 10_000, 500_000}, // 4906, 315952
		{"allocs ok", 20_000, 1_000_000},
		{"allocs passable", 50_000, 2_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 500 * time.Millisecond}, // 217.162025ms (100ms of sleep time)
		{"speed ok", 1 * time.Second},
		{"speed passable", 2 * time.Second},
	})

	// run the benchmark for 2 nodes
	res = testing.Benchmark(BenchmarkCRDTTwoNodes)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 200_000, 20_000_000}, // 137827, 10761842
		{"allocs ok", 400_000, 40_000_000},
		{"allocs passable", 750_000, 75_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 500 * time.Millisecond}, // 357.375486ms
		{"speed ok", 1 * time.Second},
		{"speed passable", 2 * time.Second},
	})

	// run the benchmark for 5 nodes
	res = testing.Benchmark(BenchmarkCRDTFiveNodes)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 300_000, 30_000_000}, // 218492, 19107136
		{"allocs ok", 500_000, 50_000_000},
		{"allocs passable", 1_000_000, 100_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 1200 * time.Millisecond}, // 744.088312ms
		{"speed ok", 2 * time.Second},
		{"speed passable", 5 * time.Second},
	})

	// run the benchmark for 10 nodes
	res = testing.Benchmark(BenchmarkCRDTTenNodes)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 750_000, 75_000_000}, // 413671, 33072456
		{"allocs ok", 1_000_000, 100_000_000},
		{"allocs passable", 1_500_000, 150_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 2 * time.Second}, // 1.38834375s
		{"speed ok", 5 * time.Second},
		{"speed passable", 10 * time.Second},
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
	runCRDT(t, 1, 1, 5000)
	runCRDT(t, 1, 1, 10000)
}

// Run runCRDT and compare results to reference assessments.
func Test_CRDT_BenchmarkCRDT_Scale_Ops(t *testing.T) {
	// run the benchmark for 100 ops
	res := testing.Benchmark(BenchmarkCRDTSmallOps)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 100_000, 5_000_000}, // 12979, 992326
		{"allocs ok", 200_000, 10_000_000},
		{"allocs passable", 500_000, 20_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 500 * time.Millisecond}, // 233.449358ms
		{"speed ok", 1 * time.Second},
		{"speed passable", 2 * time.Second},
	})

	// run the benchmark for 1000 ops
	res = testing.Benchmark(BenchmarkCRDTMediumOps)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 500_000, 50_000_000}, // 129164, 18441812
		{"allocs ok", 750_000, 75_000_000},
		{"allocs passable", 1_000_000, 100_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 1 * time.Second}, // 254.217725ms
		{"speed ok", 2 * time.Second},
		{"speed passable", 5 * time.Second},
	})

	// run the benchmark for 5000 ops
	res = testing.Benchmark(BenchmarkCRDTMediumLargeOps)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 1_000_000, 500_000_000}, // 652594, 299582392
		{"allocs ok", 2_000_000, 1_000_000_000},
		{"allocs passable", 10_000_000, 5_000_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 2 * time.Second}, // 870.914145ms
		{"speed ok", 5 * time.Second},
		{"speed passable", 10 * time.Second},
	})

	// run the benchmark for 10000
	res = testing.Benchmark(BenchmarkCRDTLargeOps)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 5_000_000, 2_000_000_000}, // 1327993, 1106469472
		{"allocs ok", 10_000_000, 5_000_000_000},
		{"allocs passable", 15_000_000, 10_000_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 10 * time.Second}, // 1.735723291s
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

// Benchmark for 5000 ops
func BenchmarkCRDTMediumLargeOps(b *testing.B) {
	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDT(b, 1, b.N, 5000)
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
