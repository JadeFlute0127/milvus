package tasks

import (
	"context"

	"github.com/milvus-io/milvus/internal/proto/querypb"
	"github.com/milvus-io/milvus/internal/querynodev2/segments"
	"github.com/milvus-io/milvus/internal/util/streamrpc"
)

var _ Task = &QueryStreamTask{}

func NewQueryStreamTask(ctx context.Context,
	collection *segments.Collection,
	manager *segments.Manager,
	req *querypb.QueryRequest,
	srv streamrpc.QueryStreamServer,
) *QueryStreamTask {
	return &QueryStreamTask{
		ctx:            ctx,
		collection:     collection,
		segmentManager: manager,
		req:            req,
		srv:            srv,
		notifier:       make(chan error, 1),
	}
}

type QueryStreamTask struct {
	ctx            context.Context
	collection     *segments.Collection
	segmentManager *segments.Manager
	req            *querypb.QueryRequest
	srv            streamrpc.QueryStreamServer
	notifier       chan error
}

// Return the username which task is belong to.
// Return "" if the task do not contain any user info.
func (t *QueryStreamTask) Username() string {
	return t.req.Req.GetUsername()
}

// PreExecute the task, only call once.
func (t *QueryStreamTask) PreExecute() error {
	return nil
}

func (t *QueryStreamTask) Execute() error {
	retrievePlan, err := segments.NewRetrievePlan(
		t.collection,
		t.req.Req.GetSerializedExprPlan(),
		t.req.Req.GetMvccTimestamp(),
		t.req.Req.Base.GetMsgID(),
	)
	if err != nil {
		return err
	}
	defer retrievePlan.Delete()

	segments, err := segments.RetrieveStream(t.ctx, t.segmentManager, retrievePlan, t.req, t.srv)
	defer t.segmentManager.Segment.Unpin(segments)
	if err != nil {
		return err
	}
	return nil
}

func (t *QueryStreamTask) Done(err error) {
	t.notifier <- err
}

func (t *QueryStreamTask) Canceled() error {
	return t.ctx.Err()
}

func (t *QueryStreamTask) Wait() error {
	return <-t.notifier
}

func (t *QueryStreamTask) NQ() int64 {
	return 1
}
