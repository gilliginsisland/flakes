package dropboxremote

import (
	"context"
	"fmt"
	"sync"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"golang.org/x/sync/errgroup"
)

const (
	uploadBatchSize = 1000
	uploadWorkers   = 20
)

type uploadObject struct {
	index int
	sha   string
	data  []byte
}

func (h *helper) uploadObjects(ctx context.Context, ids []string) error {
	for start := 0; start < len(ids); start += uploadBatchSize {
		end := min(start+uploadBatchSize, len(ids))
		entries := make([]*files.UploadSessionFinishArg, end-start)
		group, uploadCtx := errgroup.WithContext(ctx)
		queue := make(chan uploadObject, 1)
		group.Go(func() error {
			defer close(queue)
			return h.git.streamObjects(uploadCtx, ids[start:end], queue)
		})
		var progressMu sync.Mutex
		staged := start
		h.progress("Staging", staged, len(ids), false)
		for range min(uploadWorkers, end-start) {
			group.Go(func() error {
				for object := range queue {
					objectPath, err := h.objectPath(object.sha)
					if err != nil {
						return err
					}
					entry, err := h.store.stageObject(uploadCtx, objectPath, object.data)
					if err != nil {
						return err
					}
					entries[object.index] = entry
					progressMu.Lock()
					staged++
					h.progress("Staging", staged, len(ids), false)
					progressMu.Unlock()
				}
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return err
		}
		if h.showProgress {
			fmt.Fprintf(h.errOut, "\nCommitting %d objects...\n", len(entries))
		}
		if err := h.store.commitObjects(ctx, entries); err != nil {
			return err
		}
		h.progress("Writing", end, len(ids), end == len(ids))
	}
	return nil
}
