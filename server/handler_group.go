package server

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/cloud9-tools/cloud9/repo"
)

var (
	reGroupIdPath      = regexp.MustCompile(`^/group/([0-9]+)$`)
	reGroupNamePath    = regexp.MustCompile(`^/group/([A-Za-z][0-9A-Za-z]*)$`)
	reGroupName        = regexp.MustCompile(`^[A-Za-z][0-9A-Za-z]*$`)
	reGroupDescription = regexp.MustCompile(`^[\pL\pM\pN\pP\pS\pZ]*$`)
)

type Group struct {
	Id          uint64   `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	GroupName   string   `protobuf:"bytes,2,opt,name=group_name" json:"group_name,omitempty"`
	Description string   `protobuf:"bytes,3,opt,name=description" json:"description,omitempty"`
	Users       []uint64 `protobuf:"varint,4,rep,name=users" json:"users,omitempty"`
}

func (m *Group) Reset()         { *m = Group{} }
func (m *Group) String() string { return proto.CompactTextString(m) }
func (*Group) ProtoMessage()    {}

type GroupLifetime bool

const (
	NewGroup      GroupLifetime = false
	ExistingGroup GroupLifetime = true
)

type GroupDelta struct {
	GroupName   *string   `json:"group_name"`
	Description *string   `json:"description"`
	Users       *[]uint64 `json:"users"`
}

func (d *GroupDelta) Validate(lifetime GroupLifetime) error {
	if lifetime == NewGroup && d.GroupName == nil {
		return errors.New("Field 'group_name' must be set")
	}
	if d.GroupName != nil {
		switch {
		case lifetime == ExistingGroup:
			return errors.New("Field 'group_name' cannot be changed")
		case *d.GroupName == "":
			return errors.New("Field 'group_name' must be set")
		case !reGroupName.MatchString(*d.GroupName):
			return errors.New("Field 'group_name' must start with a letter and consist of letters and numbers")
		}
	}
	if d.Description != nil {
		switch {
		case *d.Description == "":
			// pass
		case !reGroupDescription.MatchString(*d.Description):
			return errors.New("Field 'description' must not contain control characters")
		}
	}
	if d.Users != nil {
		for _, id := range *d.Users {
			if id == 0 {
				return errors.New("Field 'users' must contain valid user IDs")
			}
		}
	}
	return nil
}

func (d *GroupDelta) Apply(g *Group) {
	if d.GroupName != nil {
		g.GroupName = *d.GroupName
	}
	if d.Description != nil {
		g.Description = *d.Description
	}
	if d.Users != nil {
		tmp := make([]uint64, len(*d.Users))
		copy(tmp, *d.Users)
		g.Users = tmp
	}
}

type GroupHandler struct{ Repo *repo.Repo }

func (h GroupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/group" || r.URL.Path == "/group/" {
		if !AllowMethods(w, r, GET, POST) {
			return
		}
		method := strings.ToUpper(r.Method)
		switch {
		case method == GET || method == HEAD:
			h.ListGroups(w, r)

		case method == POST:
			h.CreateGroup(w, r)

		default:
			log.Printf("error: not implemented: %s %s", method, r.URL.Path)
			http.Error(w, "Internal Server Error", 500)
		}
		return
	}

	var groupId uint64
	var groupName string
	if m := reGroupIdPath.FindStringSubmatch(r.URL.Path); m != nil {
		var err error
		groupId, err = strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			log.Printf("error: ParseUint %q 10 64: %v\n", m[1], err)
			http.NotFound(w, r)
			return
		}
	}
	if m := reGroupNamePath.FindStringSubmatch(r.URL.Path); m != nil {
		groupName = m[1]
	}
	if groupId == 0 && groupName == "" {
		http.NotFound(w, r)
		return
	}
	if !AllowMethods(w, r, GET, PUT, DELETE) {
		return
	}
	method := strings.ToUpper(r.Method)
	switch {
	case method == GET || method == HEAD:
		h.GetGroup(w, r, groupId, groupName)

	case method == PUT:
		h.PutGroup(w, r, groupId, groupName)

	case method == DELETE:
		h.DeleteGroup(w, r, groupId, groupName)

	default:
		log.Printf("error: not implemented: %s %s", method, r.URL.Path)
		http.Error(w, "Internal Server Error", 500)
	}
}

func (h GroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groupList := make([]Group, 0)
	err := h.Repo.View(repo.GROUP, func(tx *repo.Tx) error {
		return tx.ForEach(func(_ uint64, raw []byte) error {
			var g Group
			MustUnmarshalProto(raw, &g)
			groupList = append(groupList, g)
			return nil
		})
	})
	if err != nil {
		log.Printf("error: GET /group: %v\n", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	raw := MustMarshalJSON(groupList)
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlPublic)
	w.Header().Set(ETag, ETagFor(raw))
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(raw))
}

func (h GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var delta GroupDelta
	if !GetJSONBody(w, r, &delta) {
		return
	}
	if err := delta.Validate(NewGroup); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var g Group
	delta.Apply(&g)
	err := h.Repo.Update(repo.GROUP, func(tx *repo.Tx) error {
		var err error
		g.Id, err = tx.AllocateId()
		if err != nil {
			return err
		}
		err = tx.Associate(g.Id, g.GroupName)
		if err != nil {
			return err
		}
		return tx.Put(g.Id, MustMarshalProto(&g))
	})
	if _, ok := err.(*repo.DuplicateError); ok {
		http.Error(w, "There is already a group with that name.", 409)
		return
	}
	if err != nil {
		log.Printf("error: POST /group: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	raw := MustMarshalJSON(&g)
	w.Header().Set(ContentLength, fmt.Sprintf("%d", len(raw)))
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlNoCache)
	w.Header().Set(ETag, ETagFor(raw))
	w.Header().Set(Location, fmt.Sprintf("/group/%s", g.GroupName))
	w.WriteHeader(201)
	w.Write(raw)
}

func (h GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request, groupId uint64, groupName string) {
	var g Group
	err := h.Repo.View(repo.GROUP, func(tx *repo.Tx) error {
		var err error
		if groupId == 0 {
			groupId, err = tx.Lookup(groupName)
			if err != nil {
				return err
			}
		}
		value, err := tx.Get(groupId)
		if err != nil {
			return err
		}
		MustUnmarshalProto(value, &g)
		return nil
	})
	if _, ok := err.(*repo.NotFoundError); ok {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("error: GET /group %d %q: %v\n", groupId, groupName, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	raw := MustMarshalJSON(&g)
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlPublic)
	w.Header().Set(ETag, ETagFor(raw))
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(raw))
}

func (h GroupHandler) PutGroup(w http.ResponseWriter, r *http.Request, groupId uint64, groupName string) {
	var delta GroupDelta
	if !GetJSONBody(w, r, &delta) {
		return
	}
	if err := delta.Validate(ExistingGroup); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var g Group
	var done bool
	err := h.Repo.Update(repo.GROUP, func(tx *repo.Tx) error {
		var err error
		if groupId == 0 {
			groupId, err = tx.Lookup(groupName)
			if err != nil {
				return err
			}
		}
		value, err := tx.Get(groupId)
		if err != nil {
			return err
		}
		MustUnmarshalProto(value, &g)
		actualETag := ETagFor(MustMarshalJSON(&g))
		expectETag := r.Header.Get(IfMatch)
		if expectETag == "" {
			w.Header().Set(ETag, actualETag)
			http.Error(w, "Header 'If-Match' is required", 428)
			done = true
			return nil
		}
		if expectETag != actualETag {
			w.Header().Set(ETag, actualETag)
			http.Error(w, "ETag mismatch", 412)
			done = true
			return nil
		}
		delta.Apply(&g)
		return tx.Put(groupId, MustMarshalProto(&g))
	})
	if _, ok := err.(*repo.NotFoundError); ok {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("error: PUT /group %d %q: %v\n", groupId, groupName, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	if done {
		return
	}
	raw := MustMarshalJSON(&g)
	w.Header().Set(ContentLength, fmt.Sprintf("%d", len(raw)))
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlNoCache)
	w.Header().Set(ETag, ETagFor(raw))
	w.WriteHeader(200)
	w.Write(raw)
}

func (h GroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request, groupId uint64, groupName string) {
	err := h.Repo.Update(repo.GROUP, func(tx *repo.Tx) error {
		var err error
		if groupId == 0 {
			groupId, err = tx.Lookup(groupName)
			if err != nil {
				return err
			}
		}
		value, err := tx.Get(groupId)
		if err != nil {
			return err
		}
		var g Group
		MustUnmarshalProto(value, &g)
		err = tx.Delete(groupId)
		if err != nil {
			return err
		}
		return tx.Unassociate(g.GroupName)
	})
	if _, ok := err.(*repo.NotFoundError); ok {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("error: DELETE /group %d %q: %v\n", groupId, groupName, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	w.Header().Set(ContentLength, "0")
	w.WriteHeader(204)
}
