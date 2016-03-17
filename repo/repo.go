package repo

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/boltdb/bolt"
)

type ObjectType string

const (
	BLOB ObjectType = "blob"
	USER ObjectType = "user"
	GROUP ObjectType = "group"
)

var requiredBuckets = []string{
	"blob",
	"user",
	"user.byname",
	"group",
	"group.byname",
}

func (ot ObjectType) String() string {
	return string(ot)
}

func (ot ObjectType) GoString() string {
	return fmt.Sprintf("repo.ObjectType(%q)", string(ot))
}

type NotFoundError struct {
	Type ObjectType
	Id   uint64
	Name string
}

func (err *NotFoundError) Error() string {
	if err.Id != 0 {
		return fmt.Sprintf("github.com/cloud9-tools/cloud9/repo: %s not found: id %d", err.Type, err.Id)
	} else {
		return fmt.Sprintf("github.com/cloud9-tools/cloud9/repo: %s not found: name %q", err.Type, err.Name)
	}
}

type DuplicateError struct {
	Type        ObjectType
	ExistingId  uint64
	DesiredName string
}

func (err *DuplicateError) Error() string {
	return fmt.Sprintf("github.com/cloud9-tools/cloud9/repo: duplicate %s: wanted name %q, but id %d already has that name", err.Type, err.DesiredName, err.ExistingId)
}

type Repo struct {
	db *bolt.DB
}

func Open(dir string) (*Repo, error) {
	path := filepath.Join(dir, "meta.db")
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucketName := range requiredBuckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Repo{db}, nil
}

func (r *Repo) Close() error {
	return r.db.Close()
}

func (r *Repo) View(ot ObjectType, fn func(*Tx) error) error {
	return r.db.View(func (bolttx *bolt.Tx) error {
		tx := Tx{r, bolttx, ot}
		return fn(&tx)
	})
}

func (r *Repo) Update(ot ObjectType, fn func(*Tx) error) error {
	return r.db.Update(func (bolttx *bolt.Tx) error {
		tx := Tx{r, bolttx, ot}
		return fn(&tx)
	})
}

type Tx struct {
	repo *Repo
	bolttx *bolt.Tx
	ot ObjectType
}

func (tx *Tx) ForEach(fn func(uint64, []byte) error) error {
	b := tx.bolttx.Bucket([]byte(tx.ot))
	return b.ForEach(func(k, v []byte) error {
		id := btou64(k)
		return fn(id, v)
	})
}

func (tx *Tx) AllocateId() (uint64, error) {
	b := tx.bolttx.Bucket([]byte(tx.ot))
	return b.NextSequence()
}

func (tx *Tx) Get(id uint64) ([]byte, error) {
	b := tx.bolttx.Bucket([]byte(tx.ot))
	k := u64tob(id)
	v := b.Get(k)
	if v == nil {
		return nil, &NotFoundError{Type: tx.ot, Id: id}
	}
	return v, nil
}

func (tx *Tx) Put(id uint64, v []byte) error {
	b := tx.bolttx.Bucket([]byte(tx.ot))
	k := u64tob(id)
	return b.Put(k, v)
}

func (tx *Tx) Delete(id uint64) error {
	b := tx.bolttx.Bucket([]byte(tx.ot))
	k := u64tob(id)
	if v := b.Get(k); v == nil {
		return &NotFoundError{Type: tx.ot, Id: id}
	}
	return b.Delete(k)
}

func (tx *Tx) Lookup(name string) (uint64, error) {
	lcname := strings.ToLower(name)
	b := tx.bolttx.Bucket([]byte(string(tx.ot) + ".byname"))
	k := b.Get([]byte(lcname))
	if k == nil {
		return 0, &NotFoundError{Type: tx.ot, Name: name}
	}
	id := btou64(k)
	return id, nil
}

func (tx *Tx) Associate(id uint64, name string) error {
	lcname := strings.ToLower(name)
	b := tx.bolttx.Bucket([]byte(string(tx.ot) + ".byname"))
	k := b.Get([]byte(lcname))
	if k != nil {
		existingId := btou64(k)
		return &DuplicateError{Type: tx.ot, ExistingId: existingId, DesiredName: name}
	}
	k = u64tob(id)
	return b.Put([]byte(lcname), k)
}

func (tx *Tx) Unassociate(name string) error {
	lcname := strings.ToLower(name)
	b := tx.bolttx.Bucket([]byte(string(tx.ot) + ".byname"))
	return b.Delete([]byte(lcname))
}

// u64tob returns an 8-byte big endian representation of v.
func u64tob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func btou64(b []byte) uint64 {
	if len(b) < 8 {
		newB := make([]byte, 8)
		copy(newB[8-len(b):], b)
		b = newB
	}
	if len(b) > 8 {
		panic(fmt.Sprintf("btou64: len(b) = %d is > 8", len(b)))
	}
	return binary.BigEndian.Uint64(b)
}
