package taskpool

/*
实现一个按权重排序的列表容器
该容器实现按权重大的排在前面
使用下面代码创建容器:

	list:= NewPowerList() *PowerList


使用下面代码插入一个元素:

	list.Insert(lvl int64, value interface{})

使用下面代码弹出一个元素

	Pop() (elem *PowerElem)

该容器是线程不安全的，不能多线程使用.
实现原理:
实现了一个元素的单向列表
每一元素指向下一个元素,同时每一个元素都有指向自身权重的索引标记
权重索引标记同一权重最后一个元素
权重索引标记是一个单向列表,按权重大小排序,大的在前小的在后
插入元素时,判断为空,则插入第一个元素,生产索引
不为空取出第一个元素,找到该元素的权重索引,比对权重
权重相同则插入权重索引指向的元素后面
权重大于第一个元素插入倒第一个元素位置
权重小于第一个元素,则取下一个权重索引,直到找到一个权重核当前元素权重相同或小于当前权重
相同则插入到权重索引指向的元素后面,修改权重索引指向当前元素
小于则去之前的一个权重,将元素插入到这个权重索引指向元素的后面,创建当前元素的权重索引,插入倒权重索引中
*/
type PowerLister interface {
	//插入一个元素
	//lvl为权重权重大的排在前面,权重一个样后插入的在后面先插入的在前面
	Insert(lvl int64, value interface{})
	//弹出第一个元素
	Pop() (elem *PowerElem)
	//总的元素数量
	Len() int
	//复制目标容器的内容到当前容器
	Copy(PowerLister)
}

func NewPowerList() PowerLister {
	return &PowerList{}
}

type PowerMark struct {
	lvl  int64
	elem *PowerElem
	next *PowerMark
}

type PowerElem struct {
	lvl  int64
	data interface{}
	next *PowerElem
	mark *PowerMark
}

//当前权重索引标记项
func (e *PowerElem) PowerMark() *PowerMark {
	return e.mark
}
func (e *PowerElem) Lvl() int64 {
	return e.lvl
}
func (e *PowerElem) Get() interface{} {
	return e.data
}
func (e *PowerElem) Set(lvl int64, value interface{}) {
	e.lvl = lvl
	e.data = value
}
func (e *PowerElem) Next() *PowerElem {
	return e.next
}

//权重列表
type PowerList struct {
	size int
	root PowerElem
}

func (s *PowerList) First() *PowerElem {
	return s.root.next
}
func (s *PowerList) Len() int {
	return s.size
}
func (s *PowerList) insertFirst(elem *PowerElem) {
	s.size++
	next := s.root.next
	if next == nil {
		s.root.next = elem
		elem.mark = &PowerMark{lvl: elem.lvl, elem: elem}
		return
	}

	//插入单向链表 ，next指针设置
	elem.next, s.root.next = next, elem
	//创建新的权重标识
	elem.mark = &PowerMark{lvl: elem.lvl, elem: elem, next: next.mark}
}

func (s *PowerList) insertAfter(elem *PowerElem, mark *PowerElem) {
	s.size++
	//插入单向链表 ，next指针设置
	elem.next, mark.next = mark.next, elem
	if elem.lvl == mark.lvl {
		//权重相同设置权重索引
		mark.mark.elem = elem
		elem.mark = mark.mark
		return
	}
	//权重不同创建新的权重
	elem.mark = &PowerMark{lvl: elem.lvl, elem: elem, next: mark.mark.next}
	//插入权重单向链表
	mark.mark.next = elem.mark
}

//插入一个元素
func (s *PowerList) Insert(lvl int64, value interface{}) {
	//创建列表元素
	elem := &PowerElem{lvl: lvl, data: value}
	//取出根元素
	curelem := s.root.next
	if curelem == nil {
		//如果为空创建第一个元素
		s.insertFirst(elem)

		return
	}
	if curelem.Lvl() < lvl {
		//如果要插入元素权重高于第一个元素的级别则插入到最前面
		s.insertFirst(elem)
		return
	}
	curmark := curelem.mark
	for curmark.lvl > lvl {
		nextmark := curmark.next
		if nextmark == nil || nextmark.lvl < lvl {
			break
		}
		curmark = nextmark
	}
	s.insertAfter(elem, curmark.elem)

}
func (s *PowerList) Pop() (elem *PowerElem) {
	elem = s.root.next
	if elem != nil {
		s.size -= 1
		s.root.next = s.root.next.next
	}
	return
}
func (s *PowerList) Copy(list PowerLister) {
	srcs, ok := list.(*PowerList)
	if !ok {
		panic("类型不一致不能复制")
	}
	s.root = srcs.root
	s.size = srcs.size
}
