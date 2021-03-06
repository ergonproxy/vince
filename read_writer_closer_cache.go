package main

import (
	"container/list"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type readWriterCloserCache struct {
	opts   readWriterCloserCacheOption
	list   *list.List
	mu     sync.Mutex
	uses   *sync.Map
	hash   map[string]*list.Element
	opener func(string) (io.ReadWriteCloser, error)
}

type readWriterCloserCacheOption struct {
	on       boolValue
	max      intValue
	inactive durationValue
	valid    durationValue
	min      intValue
}

func (o *readWriterCloserCacheOption) defaults() {
	o.on.store(true)
	o.max.store(100)
	o.inactive.store(10 * time.Second)
	o.valid.store(60 * time.Second)
	o.min.store(1)
}

func (f *readWriterCloserCache) deleteUnsafe(node *list.Element) {
	if node == nil {
		return
	}
	v := node.Value.(*list.Element).Value.(*fileObject)
	v.file.Close()
	delete(f.hash, v.path)
	f.uses.Delete(v.path)
	f.list.Remove(node)
}

func (f *readWriterCloserCache) manageKeys(ctx context.Context) {
	valid := time.NewTicker(f.opts.valid.value)
	defer valid.Stop()
	inactive := time.NewTicker(f.opts.inactive.value)
	defer inactive.Stop()
	min := f.opts.min.value
	for {
		select {
		case <-ctx.Done():
			return
		case <-inactive.C:
			var junk []string
			f.uses.Range(func(key, value interface{}) bool {
				if value.(int64) < min {
					junk = append(junk, key.(string))
					f.uses.Delete(key)
				}
				return true
			})
			f.Delete(junk...)
		case <-valid.C:
			var junk []string
			f.uses.Range(func(key, value interface{}) bool {
				_, err := os.Stat(key.(string))
				if os.IsNotExist(err) {
					junk = append(junk, key.(string))
					f.uses.Delete(key)
				}
				return true
			})
			f.Delete(junk...)
		}
	}
}

func (f *readWriterCloserCache) Delete(keys ...string) {
	if len(keys) > 0 {
		f.mu.Lock()
		for _, key := range keys {
			f.deleteUnsafe(f.hash[key])
		}
		f.mu.Unlock()
	}
}

func (f *readWriterCloserCache) init(ctx context.Context, opts readWriterCloserCacheOption) bool {
	if !opts.on.value {
		return false
	}
	f.list = new(list.List)
	f.uses = new(sync.Map)
	f.opts = opts
	f.hash = make(map[string]*list.Element)
	go f.manageKeys(ctx)
	f.opener = func(path string) (io.ReadWriteCloser, error) {
		return os.Open(path)
	}
	return true
}

func (f *readWriterCloserCache) initFile(ctx context.Context, opts readWriterCloserCacheOption) bool {
	if ok := f.init(ctx, opts); !ok {
		return ok
	}
	f.opener = func(path string) (io.ReadWriteCloser, error) {
		return os.Open(path)
	}
	return true
}

type fileObject struct {
	path string
	file io.ReadWriteCloser
}

func (f *readWriterCloserCache) Get(key string) (io.ReadWriteCloser, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	node, ok := f.hash[key]
	if !ok {
		return nil, ok
	}
	v := node.Value.(*list.Element).Value.(*fileObject)
	f.list.MoveToFront(node)
	f.hit(key)
	return v.file, true
}

func (f *readWriterCloserCache) hit(path string) {
	if v, ok := f.uses.Load(path); ok {
		f.uses.Store(path, v.(int64)+1)
	} else {
		f.uses.Store(path, int64(1))
	}
}

func (f *readWriterCloserCache) Put(path string) (io.WriteCloser, error) {
	file, err := f.opener(path)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	if node, ok := f.hash[path]; ok {
		f.list.MoveToFront(node)
		v := node.Value.(*list.Element).Value.(*fileObject)
		node.Value.(*list.Element).Value = &fileObject{path: path, file: file}
		v.file.Close()
	} else {
		if f.list.Len() >= int(f.opts.max.value) {
			f.deleteUnsafe(f.list.Back())
		}
		f.hash[path] = f.list.PushFront(&list.Element{
			Value: &fileObject{path: path, file: file},
		})
	}
	f.hit(path)
	return file, nil
}

func (f *readWriterCloserCache) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var errs []string
	for e := f.list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*list.Element).Value.(*fileObject)
		if err := v.file.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	f.list = new(list.List)
	f.hash = make(map[string]*list.Element)
	if len(errs) > 0 {
		return errors.New("filecache: " + strings.Join(errs, ","))
	}
	return nil
}
