package gossip

import (
	"sync"

	"github.com/iotaledger/goshimmer/packages/daemon"
	"github.com/iotaledger/goshimmer/packages/events"
	"github.com/iotaledger/goshimmer/packages/node"
	"github.com/iotaledger/goshimmer/packages/transaction"
)

// region plugin module setup //////////////////////////////////////////////////////////////////////////////////////////

func configureSendQueue(plugin *node.Plugin) {
	for _, neighbor := range GetNeighbors() {
		setupEventHandlers(neighbor)
	}

	Events.AddNeighbor.Attach(events.NewClosure(setupEventHandlers))

	daemon.Events.Shutdown.Attach(events.NewClosure(func() {
		plugin.LogInfo("Stopping Send Queue Dispatcher ...")
	}))
}

func runSendQueue(plugin *node.Plugin) {
	plugin.LogInfo("Starting Send Queue Dispatcher ...")

	daemon.BackgroundWorker(func() {
		plugin.LogSuccess("Starting Send Queue Dispatcher ... done")

		for {
			select {
			case <-daemon.ShutdownSignal:
				plugin.LogSuccess("Stopping Send Queue Dispatcher ... done")

				return

			case tx := <-sendQueue:
				connectedNeighborsMutex.RLock()
				for _, neighborQueue := range neighborQueues {
					select {
					case neighborQueue.queue <- tx:
						// log sth

					default:
						// log sth
					}
				}
				connectedNeighborsMutex.RUnlock()
			}
		}
	})

	connectedNeighborsMutex.Lock()
	for _, neighborQueue := range neighborQueues {
		startNeighborSendQueue(neighborQueue)
	}
	connectedNeighborsMutex.Unlock()
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region public api ///////////////////////////////////////////////////////////////////////////////////////////////////

func SendTransaction(transaction *transaction.Transaction) {
	sendQueue <- transaction
}

func (neighbor *Neighbor) SendTransaction(transaction *transaction.Transaction) {
	if queue, exists := neighborQueues[neighbor.Identity.StringIdentifier]; exists {
		select {
		case queue.queue <- transaction:
			return

		default:
			return
		}
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region utility methods //////////////////////////////////////////////////////////////////////////////////////////////

func setupEventHandlers(neighbor *Neighbor) {
	neighbor.Events.ProtocolConnectionEstablished.Attach(events.NewClosure(func(protocol *protocol) {
		queue := &neighborQueue{
			protocol:       protocol,
			queue:          make(chan *transaction.Transaction, SEND_QUEUE_SIZE),
			disconnectChan: make(chan int, 1),
		}

		connectedNeighborsMutex.Lock()
		neighborQueues[neighbor.Identity.StringIdentifier] = queue
		connectedNeighborsMutex.Unlock()

		protocol.Conn.Events.Close.Attach(events.NewClosure(func() {
			close(queue.disconnectChan)

			connectedNeighborsMutex.Lock()
			delete(neighborQueues, neighbor.Identity.StringIdentifier)
			connectedNeighborsMutex.Unlock()
		}))

		if daemon.IsRunning() {
			startNeighborSendQueue(queue)
		}
	}))
}

func startNeighborSendQueue(neighborQueue *neighborQueue) {
	daemon.BackgroundWorker(func() {
		for {
			select {
			case <-daemon.ShutdownSignal:
				return

			case <-neighborQueue.disconnectChan:
				return

			case tx := <-neighborQueue.queue:
				switch neighborQueue.protocol.Version {
				case VERSION_1:
					sendTransactionV1(neighborQueue.protocol, tx)
				}
			}
		}
	})
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region types and interfaces /////////////////////////////////////////////////////////////////////////////////////////

type neighborQueue struct {
	protocol       *protocol
	queue          chan *transaction.Transaction
	disconnectChan chan int
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region constants and variables //////////////////////////////////////////////////////////////////////////////////////

var neighborQueues = make(map[string]*neighborQueue)

var connectedNeighborsMutex sync.RWMutex

var sendQueue = make(chan *transaction.Transaction, SEND_QUEUE_SIZE)

const (
	SEND_QUEUE_SIZE = 500
)

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
