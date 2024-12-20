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

// This test executes the exact same function as the BenchmarkCRDTSingle below.
// Its goal is mainly to raise any error that could occur during its execution as the benchmark hides them.
func Test_CRDT_Single_Doc_Benchmark_Correctness(t *testing.T) {
	runCRDTSingle(t, 1000)
}

// Run BenchmarkCRDTSingle and compare results to reference assessments
// TODO Run as follow: make test_bench_crdt_single
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

	runCRDTSingle(b, 1000)
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

	time.Sleep(time.Millisecond * 100) // TODO

	// Generate opN random operations
	ops := make([]types.CRDTOperation, opN)
	//ops[0] = tests.CreateNewBlockOp(node1.GetAddr(), docID, blockID)[0]
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

// This test executes the exact same function as the BenchmarkCRDT below.
// Its goal is mainly to raise any error that could occur during its execution as the benchmark hides them.
func Test_CRDT_Benchmark_Correctness(t *testing.T) {
	runCRDT(t, 10, 1, 100)
}

// Run BenchmarkCRDT and compare results to reference assessments
// TODO Run as follow: make test_bench_crdt
func Test_CRDT_BenchmarkCRDT(t *testing.T) {
	// run the benchmark
	res := testing.Benchmark(BenchmarkCRDT)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 5_000, 500_000},
		{"allocs ok", 10_000, 1_000_000},
		{"allocs passable", 15_000, 2_000_000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 5 * time.Second}, // TODO
		{"speed ok", 10 * time.Second},
		{"speed passable", 15 * time.Second},
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

	runCRDT(b, 10000, b.N, 1000)
}

func runCRDT(t require.TestingT, nodeCount, rounds, opN int) {
	// run as many times as specified by rounds
	for i := 0; i < rounds; i++ {
		rand.Seed(1)

		// not enough to create the intervals
		if nodeCount == 1 {
			panic("this test is meaningless for nodeCount=1")
		}

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

		time.Sleep(time.Millisecond * 100)

		// Generate opN random operations
		ops := make([]types.CRDTOperation, opN)
		//ops[0] = tests.CreateNewBlockOp(node1.GetAddr(), docID, blockID)[0]
		for i := 0; i < opN; i++ {
			// random character
			char := string(rune(rand.Intn(26) + 97))
			ops[i] = tests.CreateInsertsFromString(char, node1.GetAddr(), docID, blockID, i)[0]
		}

		crdtMsg := types.CRDTOperationsMessage{
			Operations: op,
		}
		err = node1.SaveTransactions(crdtMsg)
		require.NoError(t, err)

		// wait proportionally to the number of messages
		for i := 0; i < nodeCount*100; i++ {
			time.Sleep(time.Microsecond)
		}

		// TODO expected document content

		// cleanup
		node1.Stop()
	}
}

// This test executes the exact same function as the BenchmarkCRDTTwoNodes below.
// Its goal is mainly to raise any error that could occur during its execution as the benchmark hides them.
func Test_CRDT_Benchmark_Correctness_Two_Nodes(t *testing.T) {
	runCRDTTwoNodes(t, 2, 1, 100)
}

// Run BenchmarkCRDTTwoNodes and compare results to reference assessments
// TODO Run as follow: make test_bench_crdt_two_nodes
func Test_CRDT_BenchmarkCRDTTwoNodes(t *testing.T) {
	// run the benchmark
	res := testing.Benchmark(BenchmarkCRDTTwoNodes)

	// assess allocation against thresholds, the performance thresholds is the allocation on GitHub
	assessAllocs(t, res, []allocThresholds{
		{"allocs great", 925_000, 260_000_000},
		{"allocs ok", 1_500_000, 400_000_000},
		{"allocs passable", 2_250_000, 593750000},
	})

	// assess execution speed against thresholds, the performance thresholds is the execution speed on GitHub
	assessSpeed(t, res, []speedThresholds{
		{"speed great", 5 * time.Second}, // TODO
		{"speed ok", 10 * time.Second},
		{"speed passable", 15 * time.Second},
	})
}

// Spam a node with CRDT messages and check that it correctly processed them. We
// are going to randomly send messages filling the interval [0,N), and [N+1,2N].
// In this case the node should process the [0, N) CRDT messages, update its Editor
// and compile the document.
func BenchmarkCRDTTwoNodes(b *testing.B) {

	// Disable outputs to not penalize implementations that make use of it
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	runCRDTTwoNodes(b, 10000, b.N, 1000)
}

func runCRDTTwoNodes(t require.TestingT, nodeCount, rounds, opN int) {
	// run as many times as specified by rounds
	for i := 0; i < rounds; i++ {
		rand.Seed(1)

		// not enough to create the intervals
		if nodeCount < 2 {
			panic("this test requires at least 2 nodes")
		}

		transp := channelFac()
		node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
		node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))

		docID := "0@" + node1.GetAddr()
		blockID := "1@" + node1.GetAddr()
		op := tests.CreateNewBlockOp(node1.GetAddr(), docID, blockID)

		crdtMgs := types.CRDTOperationsMessage{
			Operations: op,
		}
		err := node1.SaveTransactions(crdtMgs)
		require.NoError(t, err)

		time.Sleep(time.Millisecond * 100)

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

		// wait proportionally to the number of messages
		for i := 0; i < nodeCount*100; i++ {
			time.Sleep(time.Microsecond)
		}

		_, err = node2.CompileDocument(docID)
		require.NoError(t, err)

		// TODO expected document content

		// cleanup
		node1.Stop()
		node2.Stop()
	}
}
