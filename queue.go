package taskpool

/**
多分片,分片内权重排序容器
使用 queue := CreateSliceQueue(writeslicenum int, sliceelemnum int) SliceQueuer
使用 Put(lvl int64, elem interface{})放入一个元素
使用 PopBlock() SliceElemer 弹出一个元素 ,如果为空阻塞
使用Len() int查看容器内有多少各元素
**/
import (
	"fmt"
	"sync/atomic"

	"sync"
)

//分片队列
type SliceQueuer interface {
	//将数据插入队列
	Put(lvl int64, elem interface{})
	//弹出一个数据，如果没有数据则阻塞
	PopBlock() SliceElemer
	Pop() SliceElemer
	//容器内元素个数 返回-1容器不可用
	Len() int
	//关闭队列，此时所有调用PopBlock方法都返回nil
	Close()
}
type SliceElemer interface {
	Lvl() int64
	Get() interface{}
}

//创建分片有限队列
//writeslicenum 分片数量
//sliceelemnum 分片内元素数量
func CreateSliceQueue(writeslicenum int, sliceelemnum int) SliceQueuer {
	queue := &SliceQueue{}
	queue.init(writeslicenum, sliceelemnum)
	return queue
}

//分片队列选项
type SliceQueueOption struct {
	//切片元素数量
	sliceelemnum int
	//写入分片数量
	writeslicenum int
}

//分片队列
type SliceQueue struct {
	//写锁
	writelock sync.Mutex
	//写等待信号
	writewait *sync.Cond
	//读等待信号
	readwait *sync.Cond
	//写入分片组
	writeQueue []PowerLister
	//读取分片
	readQueue PowerLister

	//等待切片最大数量
	sliceelemnum int
	//写入分片数量
	writeslicenum int
	size          int64
	//是否是关闭中
	closing bool
	//运行中
	run bool
}

//初始化队列
func (q *SliceQueue) init(writeslicenum int, sliceelemnum int) {
	q.sliceelemnum = sliceelemnum
	q.writeslicenum = writeslicenum
	q.readwait = sync.NewCond(&sync.Mutex{})
	q.writewait = sync.NewCond(&sync.Mutex{})
	q.readQueue = q.createSlice()
	q.writeQueue = make([]PowerLister, 0)
	q.writeQueue = append(q.writeQueue, q.createSlice())
	q.run = true
}

//创建分片
func (q *SliceQueue) createSlice() PowerLister {
	return NewPowerList()
}
func (q *SliceQueue) Len() int {
	if q.closing || !q.run {
		return -1
	}
	return int(q.size)
}
func (q *SliceQueue) Close() {

	if !q.checkRun() {
		return
	}
	fmt.Println("关闭SliceQueue")
	q.readwait.L.Lock()
	q.readwait.Broadcast()
	q.readwait.L.Unlock()
	q.closing = true
	q.run = false
}

//放入一个元素lvl为优先级 elem为元素
func (q *SliceQueue) Put(lvl int64, elem interface{}) {
	if !q.checkRun() {
		return
	}
	q.writelock.Lock()
	defer q.writelock.Unlock()
	//当前写入分片
	wq := q.writeQueue[len(q.writeQueue)-1]

	for wq.Len() >= q.sliceelemnum && !q.checkRun() {
		//当前写入分片已经满了
		if len(q.writeQueue) >= q.writeslicenum {
			//写入分片总数已经满了
			q.writewait.L.Lock()
			if len(q.writeQueue) >= q.writeslicenum {
				//写入分片总数已经满了等待下次被唤醒
				q.writewait.Wait()
				q.writewait.L.Unlock()
				continue
			} else {
				q.writewait.L.Unlock()
			}
		}
		//创建新的分片
		wq = q.createSlice()
		q.writeQueue = append(q.writeQueue, wq)
		break
	}
	//将数据插入写入分片
	wq.Insert(lvl, elem)
	atomic.AddInt64(&q.size, 1)

	//发送一个读信号
	q.readwait.L.Lock()
	q.readwait.Signal()
	q.readwait.L.Unlock()
}

//弹出一个元素
func (q *SliceQueue) PopBlock() (elem SliceElemer) {
	if !q.checkRun() {
		fmt.Println("SliceQueue 没有运行")
		return
	}
	if q.readQueue.Len() > 0 {
		//读取分片内有数据
		elem = q.readQueue.Pop()
		atomic.AddInt64(&q.size, -1)
		return
	}
	//读取分片内没有数据就检测写分片内是否有数据
	if q.writeQueue[0].Len() > 0 {
		//写入分片内有数据
		if len(q.writeQueue) > 1 {
			//写分片组有数据有多个分片
			q.writelock.Lock()
			q.readQueue.Copy(q.writeQueue[0])
			q.writeQueue = q.writeQueue[1:]
			q.writelock.Unlock()
		} else {
			//写分片组有数据只有一个分片
			q.writelock.Lock()
			q.readQueue.Copy(q.writeQueue[0])
			q.writeQueue = append(q.writeQueue, q.createSlice())
			q.writeQueue = q.writeQueue[1:]
			q.writelock.Unlock()
		}
		//产生一个取走写切片的信号
		q.writewait.L.Lock()
		q.writewait.Signal()
		q.writewait.L.Unlock()
	} else {
		//写入分片内没有数据进入读等待

		if q.writeQueue[0].Len() == 0 {

			q.readwait.L.Lock()
			if q.writeQueue[0].Len() == 0 {
				q.readwait.Wait()
			}
			q.readwait.L.Unlock()
		}
		//解锁后弹出元素
	}
	return q.Pop()
}
func (q *SliceQueue) checkRun() bool {
	return !q.closing && q.run
}

//弹出一个元素
func (q *SliceQueue) Pop() (elem SliceElemer) {
	if !q.checkRun() {
		return
	}
	if q.readQueue.Len() > 0 {
		//读取分片内有数据
		elem = q.readQueue.Pop()
		atomic.AddInt64(&q.size, -1)
		return
	}
	//读取分片内没有数据就检测写分片内是否有数据
	if q.writeQueue[0].Len() > 0 {
		//写入分片内有数据
		if len(q.writeQueue) > 1 {
			//写分片组有数据切有多个分片
			q.writelock.Lock()
			q.readQueue.Copy(q.writeQueue[0])
			q.writeQueue = q.writeQueue[1:]
			q.writelock.Unlock()
		} else {
			//写分片组有数据切只有一个分片
			q.writelock.Lock()
			q.readQueue.Copy(q.writeQueue[0])
			q.writeQueue = append(q.writeQueue, q.createSlice())
			q.writeQueue = q.writeQueue[1:]
			q.writelock.Unlock()
		}
		//产生一个取走写切片的信号
		q.writewait.L.Lock()
		q.writewait.Signal()
		q.writewait.L.Unlock()
	} else {
		//写入分片内没有数据进入读等待

		if q.writeQueue[0].Len() == 0 {

			return nil
		}
		//解锁后弹出元素

	}
	return q.Pop()
}
