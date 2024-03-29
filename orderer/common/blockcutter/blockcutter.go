/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockcutter

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	cb "github.com/hyperledger/fabric/protos/common"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/op/go-logging"
)

const pkgLogID = "orderer/common/blockcutter"

var logger *logging.Logger

func init() {
	logger = flogging.MustGetLogger(pkgLogID)
}

type OrdererConfigFetcher interface {
	OrdererConfig() (channelconfig.Orderer, bool)
}

// Receiver defines a sink for the ordered broadcast messages
type Receiver interface {
	// Ordered should be invoked sequentially as messages are ordered
	// Each batch in `messageBatches` will be wrapped into a block.
	// `pending` indicates if there are still messages pending in the receiver. It
	// is useful for Kafka orderer to determine the `LastOffsetPersisted` of block.
	Ordered(msg *cb.Envelope) (messageBatches [][]*cb.Envelope, pending bool)
	OrderedCrosschain(msg *cb.Envelope) (messageBatches [][]*cb.Envelope, pending bool) //NEW add
	// Cut returns the current batch and starts a new one
	Cut() []*cb.Envelope
}

type receiver struct {
	sharedConfigFetcher   OrdererConfigFetcher
	pendingBatch          []*cb.Envelope
	pendingBatchSizeBytes uint32
}

// NewReceiverImpl creates a Receiver implementation based on the given configtxorderer manager
func NewReceiverImpl(sharedConfigFetcher OrdererConfigFetcher) Receiver {
	return &receiver{
		sharedConfigFetcher: sharedConfigFetcher,
	}
}

// Ordered should be invoked sequentially as messages are ordered
//
// messageBatches length: 0, pending: false
//   - impossible, as we have just received a message
// messageBatches length: 0, pending: true
//   - no batch is cut and there are messages pending
// messageBatches length: 1, pending: false
//   - the message count reaches BatchSize.MaxMessageCount
// messageBatches length: 1, pending: true
//   - the current message will cause the pending batch size in bytes to exceed BatchSize.PreferredMaxBytes.
// messageBatches length: 2, pending: false
//   - the current message size in bytes exceeds BatchSize.PreferredMaxBytes, therefore isolated in its own batch.
// messageBatches length: 2, pending: true
//   - impossible
//
// Note that messageBatches can not be greater than 2.
func (r *receiver) Ordered(msg *cb.Envelope) (messageBatches [][]*cb.Envelope, pending bool) {
	ordererConfig, ok := r.sharedConfigFetcher.OrdererConfig()
	if !ok {
		logger.Panicf("Could not retrieve orderer config to query batch parameters, block cutting is not possible")
	}
	batchSize := ordererConfig.BatchSize()

	messageSizeBytes := messageSizeBytes(msg)
	if messageSizeBytes > batchSize.PreferredMaxBytes {
		logger.Debugf("The current message, with %v bytes, is larger than the preferred batch size of %v bytes and will be isolated.", messageSizeBytes, batchSize.PreferredMaxBytes)

		// cut pending batch, if it has any messages
		if len(r.pendingBatch) > 0 {
			messageBatch := r.Cut()
			messageBatches = append(messageBatches, messageBatch)
		}

		// create new batch with single message
		messageBatches = append(messageBatches, []*cb.Envelope{msg})

		return
	}

	messageWillOverflowBatchSizeBytes := r.pendingBatchSizeBytes+messageSizeBytes > batchSize.PreferredMaxBytes

	if messageWillOverflowBatchSizeBytes {
		logger.Debugf("The current message, with %v bytes, will overflow the pending batch of %v bytes.", messageSizeBytes, r.pendingBatchSizeBytes)
		logger.Debugf("Pending batch would overflow if current message is added, cutting batch now.")
		messageBatch := r.Cut()
		messageBatches = append(messageBatches, messageBatch)
	}

	logger.Debugf("Enqueuing message into batch")
	r.pendingBatch = append(r.pendingBatch, msg)
	r.pendingBatchSizeBytes += messageSizeBytes
	pending = true

	if uint32(len(r.pendingBatch)) >= batchSize.MaxMessageCount {
		logger.Debugf("Batch size met, cutting batch")
		messageBatch := r.Cut()
		messageBatches = append(messageBatches, messageBatch)
		pending = false
	}

	return
}

//NEW add  HuBinmei
func (r *receiver) OrderedCrosschain(msg *cb.Envelope) (messageBatches [][]*cb.Envelope, pending bool) {

	messageSizeBytes := messageSizeBytes(msg)

	logger.Debugf("Enqueuing message into batch")
	r.pendingBatch = append(r.pendingBatch, msg)
	r.pendingBatchSizeBytes += messageSizeBytes  //调整缓存交易消息列表的消息字节数
	pending = true

	logger.Debugf("缓存交易消息列表消息数为：%d", len(r.pendingBatch))

	//若存在缓存交易消息列表，则切割出批量交易集合
	if len(r.pendingBatch) > 0 {
		messageBatch := r.Cut()  //切割
		messageBatches = append(messageBatches, messageBatch)
	}

	logger.Debugf("缓存交易消息列表消息数为：%d", len(r.pendingBatch))

	// create new batch with single message
	messageBatches = append(messageBatches, []*cb.Envelope{msg})

	return
}

// Cut returns the current batch and starts a new one
func (r *receiver) Cut() []*cb.Envelope {
	batch := r.pendingBatch
	r.pendingBatch = nil
	r.pendingBatchSizeBytes = 0
	return batch
}

func messageSizeBytes(message *cb.Envelope) uint32 {
	return uint32(len(message.Payload) + len(message.Signature))
}
