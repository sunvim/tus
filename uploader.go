package tus

import (
	"bytes"
	"net/http"
)

type Uploader struct {
	client     *Client
	url        string
	upload     *Upload
	offset     int64
	aborted    bool
	uploadSubs []chan Upload
	notifyChan chan bool
	header     http.Header
}

// Subscribes to progress updates.
func (u *Uploader) NotifyUploadProgress(c chan Upload) {
	u.uploadSubs = append(u.uploadSubs, c)
}

// Abort aborts the upload process.
// It doens't abort the current chunck, only the remaining.
func (u *Uploader) Abort() {
	u.aborted = true
}

// IsAborted returns true if the upload was aborted.
func (u *Uploader) IsAborted() bool {
	return u.aborted
}

// Url returns the upload url.
func (u *Uploader) Url() string {
	return u.url
}

// Offset returns the current offset uploaded.
func (u *Uploader) Offset() int64 {
	return u.offset
}

func (u *Uploader) Header() http.Header {
	return u.header
}

// Upload uploads the entire body to the server.
func (u *Uploader) Upload() error {
	for u.offset < u.upload.size && !u.aborted {
		err := u.UploadChunck()

		if err != nil {
			return err
		}
	}

	return nil
}

// UploadChunck uploads a single chunck.
func (u *Uploader) UploadChunck() error {
	data := make([]byte, u.client.Config.ChunkSize)

	_, err := u.upload.stream.Seek(u.offset, 0)

	if err != nil {
		return err
	}

	size, err := u.upload.stream.Read(data)

	if err != nil {
		return err
	}

	body := bytes.NewBuffer(data[:size])

	newOffset, header, err := u.client.uploadChunck(u.url, body, int64(size), u.offset)

	if err != nil {
		return err
	}

	u.offset = newOffset

	u.upload.updateProgress(u.offset)

	for k, v := range header {
		u.header[k] = v
	}

	u.notifyChan <- true

	return nil
}

// Waits for a signal to broadcast to all subscribers
func (u *Uploader) broadcastProgress() {
	for range u.notifyChan {
		for _, c := range u.uploadSubs {
			c <- *u.upload
		}
	}
}

// NewUploader creates a new Uploader.
func NewUploader(client *Client, url string, upload *Upload, offset int64) *Uploader {
	notifyChan := make(chan bool)
	header := make(http.Header)
	uploader := &Uploader{
		client,
		url,
		upload,
		offset,
		false,
		nil,
		notifyChan,
		header,
	}

	go uploader.broadcastProgress()

	return uploader
}
