package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/golang/protobuf/proto"
)

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func MustMarshalJSON(v interface{}) []byte {
	raw, err := json.Marshal(v)
	Must(err)
	raw = append(raw, '\r', '\n')
	return raw
}

func MustMarshalProto(v proto.Message) []byte {
	raw, err := proto.Marshal(v)
	Must(err)
	return raw
}

func MustUnmarshalJSON(raw []byte, v interface{}) {
	Must(json.Unmarshal(raw, v))
}

func MustUnmarshalProto(raw []byte, v proto.Message) {
	Must(proto.Unmarshal(raw, v))
}

func AllowMethods(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	r.Method = strings.ToUpper(r.Method)
	addHead := false
	for _, method := range methods {
		if method == r.Method {
			return true
		}
		if method == GET {
			if r.Method == HEAD {
				return true
			}
			addHead = true
		}
	}
	allow := strings.Join(methods, ", ")
	if addHead {
		allow = "HEAD, " + allow
	}
	w.Header().Set(Allow, allow)
	if r.Method == OPTIONS {
		w.WriteHeader(http.StatusOK)
		return false
	}
	msg := "requires one of: " + allow
	http.Error(w, msg, http.StatusMethodNotAllowed)
	return false
}

func IsContentType(r *http.Request, t string) bool {
	if len(r.Header[ContentType]) != 1 {
		return false
	}
	s := r.Header[ContentType][0]
	return strings.EqualFold(s, t)
}

func GetJSONBody(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if !IsContentType(r, MediaTypeJSON) {
		http.Error(w, "Unsupported Media Type", 415)
		return false
	}
	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error: failed to read request body: %v\n", err)
		http.Error(w, "Internal Server Error", 500)
		return false
	}
	err = json.Unmarshal(raw, v)
	if err != nil {
		http.Error(w, "Failed to parse JSON", 400)
		return false
	}
	return true
}
