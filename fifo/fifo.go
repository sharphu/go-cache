package fifo

import (
	"cache"
	"container/list"
)

// FIFO 缓存，不是并发安全
type fifo struct {
	// 缓存容量最大值(单位byte)
	// groupcache 使用的是最大存放 entry 个数
	maxBytes int

	// 当一个 entry 从缓存中移除时调用该回调函数，默认为 nil
	// groupcache 中的 key 是任意的可比较类型；value是interface{}
	onEvicted func(key string, value interface{})

	// 已使用的字节数，只包括值，key不算
	usedBytes int

	ll *list.List
	cache map[string]*list.Element
}

type entry struct {
	key string
	value interface{}
}

func (e *entry) Len() int {
	return cache.CalcLen(e.value)
}

// New 创建新的 cache，若maxBytes是0，表示没有容量限制
func New(maxBytes int, onEvicted func(key string, value interface{})) cache.Cache {
	return &fifo{
		maxBytes: maxBytes,
		onEvicted: onEvicted,
		ll: list.New(),
		cache: make(map[string]*list.Element),
	}
}

func (f *fifo) Set(key string, value interface{}) {
	if e, ok := f.cache[key]; ok {
		f.ll.MoveToBack(e)
		en := e.Value.(*entry)
		f.usedBytes = f.usedBytes - cache.CalcLen(en.value) + cache.CalcLen(value)
		en.value = value
		return
	}

	en := &entry{key, value}
	e := f.ll.PushBack(en)
	f.cache[key] = e

	f.usedBytes += en.Len()
	if f.maxBytes > 0 && f.usedBytes > f.maxBytes {
		f.DelOldest()
	}
}

// Get 方法会从cache中获取key对应的值，nil表示key不存在
func (f *fifo) Get(key string) interface{} {
	if e, ok := f.cache[key]; ok {
		return e.Value.(*entry).value
	}

	return nil
}

// Del 方法会从cache中删除key对应的记录
func (f *fifo) Del(key string) {
	if e, ok := f.cache[key]; ok {
		f.removeElement(e)
	}
}

// DelOldest 方法会从cache中删除最旧的纪录
func (f *fifo) DelOldest() {
	f.removeElement(f.ll.Front())
}

func (f *fifo) removeElement(e *list.Element) {
	if e == nil {
		return
	}

	f.ll.Remove(e)
	en := e.Value.(*entry)
	f.usedBytes -= en.Len()
	delete(f.cache, en.key)

	if f.onEvicted != nil {
		f.onEvicted(en.key, en.value)
	}
}

// Len 返回当前cache中的记录数
func (f *fifo) Len() int {
	return f.ll.Len()
}
