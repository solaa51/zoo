package stack

import "sync"

// Stack 栈结构
type Stack struct {
	top    *node
	length int //用int可以兼容32位系统
	lock   *sync.RWMutex
}

//数据结构
type node struct {
	value interface{}
	prev  *node
}

// NewStack 初始化一个栈结构
func NewStack() *Stack {
	return &Stack{
		top:    nil,
		length: 0,
		lock:   &sync.RWMutex{},
	}
}

// Push 插入数据
func (this *Stack) Push(value interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()

	n := &node{
		value: value,
		prev:  this.top,
	}

	this.top = n
	this.length++
}

// Pop 取出数据
func (this *Stack) Pop() interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.length == 0 {
		return nil
	}

	n := this.top
	this.top = n.prev
	this.length--

	return n.value
}

func (this *Stack) Len() int {
	return this.length
}

// Peek 查看最上面的数据是什么
func (this *Stack) Peek() interface{} {
	if this.length == 0 {
		return nil
	}
	return this.top.value
}
