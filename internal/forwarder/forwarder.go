package forwarder

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/swarm-deploy/cloud-vector/internal/store/contracts"
)

const queueSize = 500

type Forwarder struct {
	store contracts.Store

	queue chan *record
}

type record struct {
	Body []byte
}

func NewForwarder(store contracts.Store) *Forwarder {
	f := &Forwarder{
		store: store,
		queue: make(chan *record, queueSize),
	}

	go f.runWorker()

	return f
}

func (f *Forwarder) Forward(writer http.ResponseWriter, req *http.Request) {
	if len(f.queue) == queueSize {
		// skip record
		writer.WriteHeader(http.StatusOK)
		return
	}

	reqBody, err := io.ReadAll(req.Body)
	if err != nil {
		slog.Error("failed to read req body", slog.Any("err", err))
		http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_ = req.Body.Close()

	f.queue <- &record{
		Body: reqBody,
	}

	writer.WriteHeader(http.StatusOK)
}

func (f *Forwarder) Stop() {
	close(f.queue)
}

func (f *Forwarder) runWorker() {
	for rec := range f.queue {
		f.send(rec)
	}
}

const sendTimeout = time.Minute

func (f *Forwarder) send(req *record) {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()

	f.doSend(ctx, req)
}

func (f *Forwarder) doSend(ctx context.Context, req *record) {
	var logs []interface{}
	if err := json.Unmarshal(req.Body, &logs); err != nil {
		slog.Error("[proxy][handler] failed to parse JSON", slog.Any("err", err))
		return
	}

	err := f.store.Push(ctx, logs)
	if err != nil {
		slog.ErrorContext(ctx, "[proxy] failed to push logs", slog.Any("err", err))
		return
	}
}
