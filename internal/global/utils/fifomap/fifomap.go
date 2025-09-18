package fifomap

import (
	"container/list"
	"sync"
)


type FIFOMap struct {
	maxSize  int                           // 最大容量
	elements *list.List                    // 双向链表维护顺序
	items    map[interface{}]*list.Element // 快速查找
	lock     sync.RWMutex                  // 线程安全
}

type entry struct {
	key   interface{}
	value interface{}
}

func NewFIFOMap(maxSize int) *FIFOMap {
	if maxSize <= 0 {
		panic("maxSize must be positive")
	}
	return &FIFOMap{
		maxSize:  maxSize,
		elements: list.New(),
		items:    make(map[interface{}]*list.Element),
	}
}

func (m *FIFOMap) Set(key, value interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 如果key已存在，更新值并移到链表前端
	if elem, exists := m.items[key]; exists {
		m.elements.MoveToFront(elem)
		elem.Value.(*entry).value = value
		return
	}

	// 达到容量上限时淘汰最旧元素
	if m.elements.Len() >= m.maxSize {
		oldest := m.elements.Back()
		if oldest != nil {
			delete(m.items, oldest.Value.(*entry).key)
			m.elements.Remove(oldest)
		}
	}

	// 添加新元素
	elem := m.elements.PushFront(&entry{key, value})
	m.items[key] = elem
}

func (m *FIFOMap) Get(key interface{}) (interface{}, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if elem, exists := m.items[key]; exists {
		// 访问时也可选择更新位置（LRU特性，可选）
		// m.elements.MoveToFront(elem)
		return elem.Value.(*entry).value, true
	}
	return nil, false
}

func (m *FIFOMap) Delete(key interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if elem, exists := m.items[key]; exists {
		m.elements.Remove(elem)
		delete(m.items, key)
	}
}

func (m *FIFOMap) GetAll() []interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()

	result := make([]interface{}, 0, m.elements.Len())
	for elem := m.elements.Front(); elem != nil; elem = elem.Next() {
		result = append(result, elem.Value.(*entry).value)
	}
	return result
}

func (m *FIFOMap) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.elements = list.New()
	m.items = make(map[interface{}]*list.Element)
}
