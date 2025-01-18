package keythrottle

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"
)

func TestThrottle_HandleCancelledQueue(t *testing.T) {
	// Arrange
	connCustomer := BuildConnectedCustomer()

	// Act
	ctx := context.Background()
	cancelledCtx, cancel := context.WithCancel(ctx)
	connCustomer.AddRequest(cancelledCtx)
	cancel()
	time.Sleep(1 * time.Second)

	err := connCustomer.TryExecute()
	// Assert
	if err == nil {
		t.Error("Expected error, got none")
	}

}

func TestThrottle_HandleMultipleQueue(t *testing.T) {
	connCustomer := BuildConnectedCustomer()
	ctx1 := context.Background()
	ctx1, cancel1 := context.WithCancel(ctx1)
	log.Println("adding first request")
	med1 := connCustomer.AddRequest(ctx1)
	ctx2 := context.Background()
	log.Println("adding second request")
	med2 := connCustomer.AddRequest(ctx2)

	cancel1()

	time.Sleep(1 * time.Second)
	log.Println("executing 1st time")
	err := connCustomer.TryExecute()

	if err != nil {
		t.Error("Expected no error, got ", err)
	}

	log.Println("waiting on channels after 1st execution")
	select {
	case <-med1.ExecuteChan:
		t.Error("Expected no ExecuteChan, got ", med1.ExecuteChan)
	case <-med2.ExecuteChan:
		break
	case <-time.After(1 * time.Second):
		t.Error("timed out")
	}

	log.Println("executing 2nd time")
	err = connCustomer.TryExecute()
	if err == nil {
		t.Error("Expected no error, got ", err)
	}

	select {
	case <-med1.ExecuteChan:
		t.Error("did not expect req1 to execute")
	case <-med2.ExecuteChan:
		t.Error("did not expect req2 to execute")
	case <-time.After(1 * time.Second):

	}

	log.Printf("Performing final execute, still should get not found")
	err = connCustomer.TryExecute()
	if err == nil {
		t.Error("Expected error, got none")
	}
}

func TestThrottle_HandleEmptyQueue(t *testing.T) {
	connCustomer := BuildConnectedCustomer()

	err := connCustomer.TryExecute()
	var nfError RequestNotFoundError
	if !errors.As(err, &nfError) {
		t.Error("Expected Error")
	} else {
		_, ok := err.(RequestNotFoundError)
		if !ok {
			t.Error("Expected RequestNotFoundError")
		}

	}

}
