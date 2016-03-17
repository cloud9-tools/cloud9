package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloud9-tools/cloud9/repo"
)

var (
	reBlobIdPath         = regexp.MustCompile(`^/blob/([0-9]+)$`)
	reMultipartMediaType = regexp.MustCompile(`(?i)^multipart/.*$`)
)

type BlobHandler struct{ repo *repo.Repo }

type BlobReference struct {
	Id uint64 `json:"id"`
}

func (h BlobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/blob" || r.URL.Path == "/blob/" {
		if !AllowMethods(w, r, GET, POST) {
			return
		}
		method := strings.ToUpper(r.Method)
		switch {
		case method == GET || method == HEAD:
			h.ListBlobs(w, r)

		case method == POST:
			h.CreateBlob(w, r)

		default:
			log.Printf("error: not implemented: %s %s", method, r.URL.Path)
			http.Error(w, "Internal Server Error", 500)
		}
		return
	}

	m := reBlobIdPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	blobId, err := strconv.ParseUint(m[1], 10, 64)
	if err != nil {
		log.Printf("error: ParseUint %q 10 64: %v\n", m[1], err)
		http.NotFound(w, r)
		return
	}
	if !AllowMethods(w, r, GET) {
		return
	}
	h.GetBlob(w, r, blobId)
}

func (h BlobHandler) ListBlobs(w http.ResponseWriter, r *http.Request) {
	blobList := make([]BlobReference, 0)
	err := h.repo.View(repo.BLOB, func(tx *repo.Tx) error {
		return tx.ForEach(func(id uint64, _ []byte) error {
			blobList = append(blobList, BlobReference{Id: id})
			return nil
		})
	})
	if err != nil {
		log.Printf("error: GET /blob: %v\n", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	raw := MustMarshalJSON(blobList)
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlPublic)
	w.Header().Set(ETag, ETagFor(raw))
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(raw))
}

func (h BlobHandler) CreateBlob(w http.ResponseWriter, r *http.Request) {
	if len(r.Header[ContentType]) != 1 || reMultipartMediaType.MatchString(r.Header[ContentType][0]) {
		http.Error(w, "Unsupported Media Type", 415)
		return
	}
	blob, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error: failed to read request body: %v\n", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	var id uint64
	err = h.repo.Update(repo.BLOB, func(tx *repo.Tx) error {
		var err error
		id, err = tx.AllocateId()
		if err != nil {
			return err
		}
		return tx.Put(id, blob)
	})
	if err != nil {
		log.Printf("error: POST /blob: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	raw := MustMarshalJSON(BlobReference{Id: id})
	w.Header().Set(ContentLength, fmt.Sprintf("%d", len(raw)))
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlNoCache)
	w.Header().Set(ETag, ETagFor(blob))
	w.Header().Set(Location, fmt.Sprintf("/blob/%d", id))
	w.WriteHeader(201)
	w.Write(raw)
}

func (h BlobHandler) GetBlob(w http.ResponseWriter, r *http.Request, blobId uint64) {
	var blob []byte
	err := h.repo.View(repo.BLOB, func(tx *repo.Tx) error {
		var err error
		blob, err = tx.Get(blobId)
		if err != nil {
			return err
		}
		return nil
	})
	if _, ok := err.(*repo.NotFoundError); ok {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("error: GET /blob %d: %v\n", blobId, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	w.Header().Set(ContentType, MediaTypeBinary)
	w.Header().Set(CacheControl, CacheControlPublic)
	w.Header().Set(ETag, ETagFor(blob))
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(blob))
}
