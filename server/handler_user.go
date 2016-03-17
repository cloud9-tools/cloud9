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
	reUserIdPath      = regexp.MustCompile(`^/user/([0-9]+)$`)
	reUserNamePath    = regexp.MustCompile(`^/user/([A-Za-z][0-9A-Za-z]*)$`)
	reUserName        = regexp.MustCompile(`^[A-Za-z][0-9A-Za-z]*$`)
	reUserDisplayName = regexp.MustCompile(`^[\pL\pM\pN\pP\pS\pZ]+$`)
	reEMail           = regexp.MustCompile(`(?i)^[0-9a-z_.+-]+@[0-9a-z][0-9a-z_-]*(?:\.[0-9a-z][0-9a-z_-]*)+$`)
	reURL             = regexp.MustCompile(`(?i)^https?://(?:[0-9a-z][0-9a-z_-]*(?:\.[0-9a-z][0-9a-z_-]*)+|\d+\.\d+\.\d+\.\d+|\[[0-9a-f:.]+\])(?::[1-9]\d*)?(?:/\PC*)?$`)
)

type User struct {
	Id          uint64 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	UserName    string `protobuf:"bytes,2,opt,name=user_name" json:"user_name,omitempty"`
	DisplayName string `protobuf:"bytes,3,opt,name=display_name" json:"display_name,omitempty"`
	EMail       string `protobuf:"bytes,4,opt,name=email" json:"email,omitempty"`
	URL         string `protobuf:"bytes,5,opt,name=url" json:"url,omitempty"`
}

func (m *User) Reset()         { *m = User{} }
func (m *User) String() string { return proto.CompactTextString(m) }
func (*User) ProtoMessage()    {}

type UserLifetime bool

const (
	NewUser      UserLifetime = false
	ExistingUser UserLifetime = true
)

type UserDelta struct {
	UserName    *string `json:"user_name"`
	DisplayName *string `json:"display_name"`
	EMail       *string `json:"email"`
	URL         *string `json:"url"`
}

func (d *UserDelta) Validate(lifetime UserLifetime) error {
	if lifetime == NewUser && d.UserName == nil {
		return errors.New("Field 'user_name' must be set")
	}
	if lifetime == NewUser && d.EMail == nil {
		return errors.New("Field 'email' must be set")
	}
	if d.UserName != nil {
		switch {
		case lifetime == ExistingUser:
			return errors.New("Field 'user_name' cannot be changed")
		case *d.UserName == "":
			return errors.New("Field 'user_name' must be set")
		case !reUserName.MatchString(*d.UserName):
			return errors.New("Field 'user_name' must start with a letter and consist of letters and numbers")
		}
	}
	if d.DisplayName != nil {
		switch {
		case *d.DisplayName == "":
			// pass
		case !reUserDisplayName.MatchString(*d.DisplayName):
			return errors.New("Field 'display_name' must not contain control characters")
		}
	}
	if d.EMail != nil {
		switch {
		case *d.EMail == "":
			return errors.New("Field 'email' must be set")
		case !reEMail.MatchString(*d.EMail):
			return errors.New("Field 'email' must be a valid e-mail address")
		}
	}
	if d.URL != nil {
		switch {
		case *d.URL == "":
			// pass
		case !reURL.MatchString(*d.URL):
			return errors.New("Field 'url' must be a valid HTTP(S) URL")
		}
	}
	return nil
}

func (d *UserDelta) Apply(u *User) {
	if d.UserName != nil {
		u.UserName = *d.UserName
	}
	if d.DisplayName != nil {
		if *d.DisplayName == "" {
			u.DisplayName = u.UserName
		} else {
			u.DisplayName = *d.DisplayName
		}
	}
	if d.EMail != nil {
		u.EMail = *d.EMail
	}
	if d.URL != nil {
		u.URL = *d.URL
	}
}

type UserHandler struct{ Repo *repo.Repo }

func (h UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/user" || r.URL.Path == "/user/" {
		if !AllowMethods(w, r, GET, POST) {
			return
		}
		method := strings.ToUpper(r.Method)
		switch {
		case method == GET || method == HEAD:
			h.ListUsers(w, r)

		case method == POST:
			h.CreateUser(w, r)

		default:
			log.Printf("error: not implemented: %s %s", method, r.URL.Path)
			http.Error(w, "Internal Server Error", 500)
		}
		return
	}

	var userId uint64
	var userName string
	if m := reUserIdPath.FindStringSubmatch(r.URL.Path); m != nil {
		var err error
		userId, err = strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			log.Printf("error: ParseUint %q 10 64: %v\n", m[1], err)
			http.NotFound(w, r)
			return
		}
	}
	if m := reUserNamePath.FindStringSubmatch(r.URL.Path); m != nil {
		userName = m[1]
	}
	if userId == 0 && userName == "" {
		http.NotFound(w, r)
		return
	}
	if !AllowMethods(w, r, GET, PUT, DELETE) {
		return
	}
	method := strings.ToUpper(r.Method)
	switch {
	case method == GET || method == HEAD:
		h.GetUser(w, r, userId, userName)

	case method == PUT:
		h.PutUser(w, r, userId, userName)

	case method == DELETE:
		h.DeleteUser(w, r, userId, userName)

	default:
		log.Printf("error: not implemented: %s %s", method, r.URL.Path)
		http.Error(w, "Internal Server Error", 500)
	}
}

func (h UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	userList := make([]User, 0)
	err := h.Repo.View(repo.USER, func(tx *repo.Tx) error {
		return tx.ForEach(func(_ uint64, raw []byte) error {
			var u User
			MustUnmarshalProto(raw, &u)
			userList = append(userList, u)
			return nil
		})
	})
	if err != nil {
		log.Printf("error: GET /user: %v\n", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	raw := MustMarshalJSON(userList)
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlPublic)
	w.Header().Set(ETag, ETagFor(raw))
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(raw))
}

func (h UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var delta UserDelta
	if !GetJSONBody(w, r, &delta) {
		return
	}
	if err := delta.Validate(NewUser); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var u User
	delta.Apply(&u)
	err := h.Repo.Update(repo.USER, func(tx *repo.Tx) error {
		var err error
		u.Id, err = tx.AllocateId()
		if err != nil {
			return err
		}
		err = tx.Associate(u.Id, u.UserName)
		if err != nil {
			return err
		}
		return tx.Put(u.Id, MustMarshalProto(&u))
	})
	if _, ok := err.(*repo.DuplicateError); ok {
		http.Error(w, "There is already a user with that name.", 409)
		return
	}
	if err != nil {
		log.Printf("error: POST /user: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	raw := MustMarshalJSON(&u)
	w.Header().Set(ContentLength, fmt.Sprintf("%d", len(raw)))
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlNoCache)
	w.Header().Set(ETag, ETagFor(raw))
	w.Header().Set(Location, fmt.Sprintf("/user/%s", u.UserName))
	w.WriteHeader(201)
	w.Write(raw)
}

func (h UserHandler) GetUser(w http.ResponseWriter, r *http.Request, userId uint64, userName string) {
	var u User
	err := h.Repo.View(repo.USER, func(tx *repo.Tx) error {
		var err error
		if userId == 0 {
			userId, err = tx.Lookup(userName)
			if err != nil {
				return err
			}
		}
		value, err := tx.Get(userId)
		if err != nil {
			return err
		}
		MustUnmarshalProto(value, &u)
		return nil
	})
	if _, ok := err.(*repo.NotFoundError); ok {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("error: GET /user %d %q: %v\n", userId, userName, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	raw := MustMarshalJSON(&u)
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlPublic)
	w.Header().Set(ETag, ETagFor(raw))
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(raw))
}

func (h UserHandler) PutUser(w http.ResponseWriter, r *http.Request, userId uint64, userName string) {
	var delta UserDelta
	if !GetJSONBody(w, r, &delta) {
		return
	}
	if err := delta.Validate(ExistingUser); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var u User
	var done bool
	err := h.Repo.Update(repo.USER, func(tx *repo.Tx) error {
		var err error
		if userId == 0 {
			userId, err = tx.Lookup(userName)
			if err != nil {
				return err
			}
		}
		value, err := tx.Get(userId)
		if err != nil {
			return err
		}
		MustUnmarshalProto(value, &u)
		actualETag := ETagFor(MustMarshalJSON(&u))
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
		delta.Apply(&u)
		return tx.Put(userId, MustMarshalProto(&u))
	})
	if _, ok := err.(*repo.NotFoundError); ok {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("error: PUT /user %d %q: %v\n", userId, userName, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	if done {
		return
	}
	raw := MustMarshalJSON(&u)
	w.Header().Set(ContentLength, fmt.Sprintf("%d", len(raw)))
	w.Header().Set(ContentType, MediaTypeJSON)
	w.Header().Set(CacheControl, CacheControlNoCache)
	w.Header().Set(ETag, ETagFor(raw))
	w.WriteHeader(200)
	w.Write(raw)
}

func (h UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request, userId uint64, userName string) {
	err := h.Repo.Update(repo.USER, func(tx *repo.Tx) error {
		var err error
		if userId == 0 {
			userId, err = tx.Lookup(userName)
			if err != nil {
				return err
			}
		}
		value, err := tx.Get(userId)
		if err != nil {
			return err
		}
		var u User
		MustUnmarshalProto(value, &u)
		err = tx.Delete(userId)
		if err != nil {
			return err
		}
		return tx.Unassociate(u.UserName)
	})
	if _, ok := err.(*repo.NotFoundError); ok {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("error: DELETE /user %d %q: %v\n", userId, userName, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	w.Header().Set(ContentLength, "0")
	w.WriteHeader(204)
}
