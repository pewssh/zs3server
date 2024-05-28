package zcn

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
)

var (
	batchUploadChan chan sdk.OperationRequest
	workerChan      chan []sdk.OperationRequest
)

func IntiBatchUploadWorkers(ctx context.Context, alloc *sdk.Allocation, waitTime, maxOperations, maxWorkers int) {

	batchUploadChan = make(chan sdk.OperationRequest, maxOperations*maxWorkers)
	workerChan = make(chan []sdk.OperationRequest, maxWorkers)

	for i := 0; i < maxWorkers; i++ {
		go batchUploadWorker(ctx, alloc, workerChan)
	}
	opRequest := make([]sdk.OperationRequest, 0, 5)
	opWaitTime := waitTime
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case op := <-batchUploadChan:
				// process the batch upload or wait for more operations
				opRequest = append(opRequest, op)
				if len(opRequest) == maxOperations || opWaitTime >= waitTime*3 {
					log.Println("process batch for time condition")
					workerChan <- opRequest
					opRequest = make([]sdk.OperationRequest, 0, 5)
					opWaitTime = waitTime
					continue
				}
				if len(batchUploadChan) == 0 {
					// wait for more operations
					log.Println("waiting for more operations: ", opWaitTime, len(opRequest))
					time.Sleep(time.Duration(opWaitTime) * time.Millisecond)
				} else {
					// consume more operations
					continue
				}
				// if there are no more operations process the batch
				if len(batchUploadChan) == 0 {
					log.Println("process batch for no more ops condition")
					workerChan <- opRequest
					opRequest = make([]sdk.OperationRequest, 0, 5)
					opWaitTime = waitTime
				} else {
					opWaitTime += (opWaitTime / 2)
				}
			}
		}
	}()

}

func batchUploadWorker(ctx context.Context, alloc *sdk.Allocation, opsChan chan []sdk.OperationRequest) {
	for {
		select {
		case <-ctx.Done():
			return
		case ops := <-opsChan:
			// process the batch upload or wait for more operations
			log.Println("processing batch upload: ", len(ops))
			err := alloc.DoMultiOperation(ops)
			if err != nil {
				if !isSameRootError(err) {
					err = nil
				} else if err == context.Canceled {
					err = errors.New("operation canceled")
				}
			}
			for _, op := range ops {
				op.CancelCauseFunc(err)
			}
		}
	}
}
