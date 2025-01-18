package keythrottle

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
)

var nextQueuedRequestid uint64 = 0

type HandlerMediator struct {
	ExecuteChan chan struct{}
}

type QueuedRequest struct {
	ID uint64
	// executeChannel is signaled when it is time to handle the request
	executeChannel chan struct{}
	// isCancelled notes that the request has been cancelled
	isCancelled             bool
	cancelledMutex          sync.Mutex
	stoppedListeningExecute chan struct{}
}

func (qr *QueuedRequest) IsCancelled() bool {
	qr.cancelledMutex.Lock()
	defer qr.cancelledMutex.Unlock()
	log.Println("checking if cancelled:", qr.isCancelled)
	return qr.isCancelled
}

func (qr *QueuedRequest) SetCancelled() {
	qr.cancelledMutex.Lock()
	defer qr.cancelledMutex.Unlock()
	log.Println("setting cancelled:", qr.isCancelled)
	qr.isCancelled = true
}

func (q *QueuedRequest) BuildMediator() *HandlerMediator {
	m := HandlerMediator{
		ExecuteChan: q.executeChannel,
	}
	return &m
}

func (q *QueuedRequest) Init() {
	q.executeChannel = make(chan struct{}, 1)
	q.stoppedListeningExecute = make(chan struct{}, 1)
}

func (q *QueuedRequest) SignalExecute() {
	// This stops the forever go routine listening for cancellation.
	q.stoppedListeningExecute <- struct{}{}
	q.executeChannel <- struct{}{}
}

type RequestNotFoundError struct {
}

func (e RequestNotFoundError) Error() string {
	return "request not found"
}

type ConnectedCustomer struct {
	queuedRequests []*QueuedRequest
	mutex          sync.Mutex
}

func (c *ConnectedCustomer) Init() {
	c.queuedRequests = []*QueuedRequest{}
	c.mutex = sync.Mutex{}
}

func BuildConnectedCustomer() *ConnectedCustomer {
	c := ConnectedCustomer{}
	c.Init()
	return &c

}

func (c *ConnectedCustomer) AddRequest(ctx context.Context) *HandlerMediator {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	q := QueuedRequest{
		ID: atomic.AddUint64(&nextQueuedRequestid, 1),
	}
	q.Init()
	go func() {
		select {
		case <-ctx.Done():
			c.mutex.Lock()
			toDelete := -1
			for i, qr := range c.queuedRequests {
				if qr.ID == q.ID {
					toDelete = i
				}
			}
			if toDelete >= 0 {
				c.queuedRequests = removeByIndex(c.queuedRequests, toDelete)
			}
			q.SetCancelled()
			c.mutex.Unlock()
		case <-q.stoppedListeningExecute:
		}

	}()
	c.queuedRequests = append(c.queuedRequests, &q)
	m := q.BuildMediator()
	return m
}

func (c *ConnectedCustomer) TryExecute() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	foundIndex := -1
	for i, q := range c.queuedRequests {
		if !q.IsCancelled() {
			foundIndex = i
			break
		}
	}
	if foundIndex != -1 {
		foundRequest := c.queuedRequests[foundIndex]
		foundRequest.SignalExecute()
		c.queuedRequests = removeByIndex(c.queuedRequests, foundIndex)
		return nil
	}

	return RequestNotFoundError{}
}

func removeByIndex[T any](s []T, index int) []T {
	if index < 0 || index >= len(s) {
		return s // Index out of range; return the original slice
	}

	return append(s[:index], s[index+1:]...)
}
