package dropboxremote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/retry"
)

const chunkSize = 50 * 1024 * 1024

var errNotFound = errors.New("Dropbox path not found")
var errConflict = errors.New("concurrent modification detected (fetch first)")

type storage struct {
	api    files.ContextClient
	errOut io.Writer
	secret string
}

func (s storage) get(ctx context.Context, path string) (_ string, _ []byte, err error) {
	debugf(s.errOut, "Dropbox download path=%q started", path)
	defer func() { debugResult(s.errOut, fmt.Sprintf("Dropbox download path=%q", path), err, s.secret) }()
	meta, body, err := s.api.DownloadContext(ctx, files.NewDownloadArg(path))
	if err != nil {
		var e files.DownloadAPIError
		if errors.As(err, &e) && e.EndpointError != nil && e.EndpointError.Path != nil && e.EndpointError.Path.Tag == files.LookupErrorNotFound {
			return "", nil, errNotFound
		}
		return "", nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	return meta.Rev, data, err
}
func (s storage) list(ctx context.Context, path string) (_ []*files.FileMetadata, err error) {
	debugf(s.errOut, "Dropbox list path=%q started", path)
	defer func() { debugResult(s.errOut, fmt.Sprintf("Dropbox list path=%q", path), err, s.secret) }()
	arg := files.NewListFolderArg(path)
	arg.Recursive = true
	result, err := s.api.ListFolderContext(ctx, arg)
	if err != nil {
		var e files.ListFolderAPIError
		if errors.As(err, &e) && e.EndpointError != nil && e.EndpointError.Path != nil && e.EndpointError.Path.Tag == files.LookupErrorNotFound {
			return nil, errNotFound
		}
		return nil, err
	}
	var entries []*files.FileMetadata
	for {
		for _, entry := range result.Entries {
			if file, ok := entry.(*files.FileMetadata); ok {
				entries = append(entries, file)
			}
		}
		if !result.HasMore {
			return entries, nil
		}
		debugf(s.errOut, "Dropbox list continuing path=%q entries=%d", path, len(entries))
		result, err = s.api.ListFolderContinueContext(ctx, files.NewListFolderContinueArg(result.Cursor))
		if err != nil {
			return nil, err
		}
	}
}
func (s storage) put(ctx context.Context, path string, data []byte, mode *files.WriteMode) (err error) {
	debugf(s.errOut, "Dropbox upload path=%q started", path)
	defer func() { debugResult(s.errOut, fmt.Sprintf("Dropbox upload path=%q", path), err, s.secret) }()
	debugf(s.errOut, "upload bytes=%d mode=%s", len(data), mode.Tag)
	arg := files.NewUploadArg(path)
	arg.Mode, arg.StrictConflict, arg.Mute = mode, true, true
	_, err = s.api.UploadContext(ctx, arg, bytes.NewReader(data))
	var e files.UploadAPIError
	if errors.As(err, &e) && e.EndpointError != nil && e.EndpointError.Path != nil && e.EndpointError.Path.Reason != nil && e.EndpointError.Path.Reason.Tag == files.WriteErrorConflict {
		return errConflict
	}
	return err
}
func (s storage) delete(ctx context.Context, path string) (err error) {
	debugf(s.errOut, "Dropbox delete path=%q started", path)
	defer func() { debugResult(s.errOut, fmt.Sprintf("Dropbox delete path=%q", path), err, s.secret) }()
	_, err = s.api.DeleteV2Context(ctx, files.NewDeleteArg(path))
	var e files.DeleteV2APIError
	if errors.As(err, &e) && e.EndpointError != nil && e.EndpointError.PathLookup != nil && e.EndpointError.PathLookup.Tag == files.LookupErrorNotFound {
		return nil
	}
	return err
}
func writeMode(tag, rev string) *files.WriteMode {
	return &files.WriteMode{Tagged: dropbox.Tagged{Tag: tag}, Update: rev}
}
func (s storage) stageObject(ctx context.Context, path string, data []byte) (_ *files.UploadSessionFinishArg, err error) {
	debugf(s.errOut, "staging object path=%q bytes=%d", path, len(data))
	defer func() { debugResult(s.errOut, "stage object", err, s.secret) }()
	end := min(chunkSize, len(data))
	arg := files.NewUploadSessionStartArg()
	arg.Close = end == len(data)
	session, err := s.api.UploadSessionStartContext(ctx, arg, bytes.NewReader(data[:end]))
	if err != nil {
		return nil, err
	}
	cursor := files.NewUploadSessionCursor(session.SessionId, uint64(end))
	retries := 0
	for cursor.Offset < uint64(len(data)) {
		end := min(cursor.Offset+chunkSize, uint64(len(data)))
		arg := files.NewUploadSessionAppendArg(cursor)
		arg.Close = end == uint64(len(data))
		debugf(s.errOut, "upload chunk offset=%d end=%d total=%d", cursor.Offset, end, len(data))
		err := s.api.UploadSessionAppendV2Context(ctx, arg, bytes.NewReader(data[cursor.Offset:end]))
		if err == nil {
			cursor.Offset = end
			continue
		}
		var apiErr files.UploadSessionAppendV2APIError
		if !errors.As(err, &apiErr) || apiErr.EndpointError == nil || apiErr.EndpointError.IncorrectOffset == nil || retries >= 3 {
			return nil, err
		}
		offset := apiErr.EndpointError.IncorrectOffset.CorrectOffset
		if offset > uint64(len(data)) {
			return nil, fmt.Errorf("invalid upload session offset %d", offset)
		}
		cursor.Offset = offset
		retries++
	}
	commit := files.NewCommitInfo(path)
	commit.Mode, commit.StrictConflict, commit.Mute = writeMode(files.WriteModeOverwrite, ""), true, true
	return files.NewUploadSessionFinishArg(cursor, commit), nil
}

func (s storage) commitObjects(ctx context.Context, entries []*files.UploadSessionFinishArg) (err error) {
	defer func() { debugResult(s.errOut, "commit object batch", err, s.secret) }()
	for len(entries) > 0 {
		debugf(s.errOut, "committing %d objects", len(entries))
		result, err := s.api.UploadSessionFinishBatchV2Context(ctx, files.NewUploadSessionFinishBatchArg(entries))
		if err != nil {
			return err
		}
		if result == nil || len(result.Entries) != len(entries) {
			return fmt.Errorf("Dropbox batch commit returned an unexpected number of results")
		}
		var pending []*files.UploadSessionFinishArg
		for i, entry := range result.Entries {
			if entry != nil && entry.Tag == files.UploadSessionFinishBatchResultEntrySuccess {
				continue
			}
			if entry != nil && entry.Failure != nil {
				failure := entry.Failure
				if failure.Tag == files.UploadSessionFinishErrorTooManyWriteOperations ||
					(failure.Path != nil && failure.Path.Tag == files.WriteErrorTooManyWriteOperations) {
					pending = append(pending, entries[i])
					continue
				}
			}
			details, _ := json.Marshal(entry)
			return fmt.Errorf("Dropbox batch commit failed for %q: %s", entries[i].Commit.Path, details)
		}
		entries = pending
		if len(entries) > 0 {
			debugf(s.errOut, "retrying %d contended batch entries in 5 seconds", len(entries))
			if err := retry.Sleep(ctx, 5*time.Second); err != nil {
				return err
			}
		}
	}
	return nil
}
