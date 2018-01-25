package comm

import (
	"container/list"
	"sync"
)

type TSList struct {
	list      *list.List
	lockMutex sync.Mutex
}

func NewTSList() *TSList {
	return &TSList{list.New(), sync.Mutex{}}
}

func (this *TSList) PushBack(elem interface{}) {
	this.Lock()
	defer this.Unlock()

	if elem != nil {
		this.list.PushBack(elem)
	}
}

func (this *TSList) Remove(elem *list.Element) {
	this.Lock()
	defer this.Unlock()

	if elem != nil {
		this.list.Remove(elem)
	}
}

func (this *TSList) Pop() (interface{}, bool) {
	this.Lock()
	defer this.Unlock()

	if this.list.Len() == 0 {
		return nil, false
	}

	element := this.list.Back()
	this.list.Remove(element)

	return element.Value, true
}

func (this *TSList) Len() int {
	this.Lock()
	defer this.Unlock()

	return this.list.Len()
}

func (this *TSList) IsEmpty() bool {
	this.Lock()
	defer this.Unlock()

	return this.list.Len() == 0
}

func (this *TSList) Front() *list.Element {
	this.Lock()
	defer this.Unlock()

	return this.list.Front()
}

func (this *TSList) Back() *list.Element {
	this.Lock()
	defer this.Unlock()

	return this.list.Back()
}

func (this *TSList) Lock() {
	this.lockMutex.Lock()
}

func (this *TSList) Unlock() {
	this.lockMutex.Unlock()
}
