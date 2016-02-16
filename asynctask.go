package taskpool

import (
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"sync/atomic"
)

type Task interface {
	Failded(err error)
	GetFaildedNum() uint64
	Success(result ...interface{})
	faidedAdd()
}
type BaseTask struct {
	faild uint64
}

func (t *BaseTask) Failded(err error) {
}
func (t *BaseTask) faidedAdd() {
	atomic.AddUint64(&t.faild, 1)
}

func (t *BaseTask) GetFaildedNum() uint64 {
	return t.faild
}
func (t *BaseTask) Success(result ...interface{}) {

}

type AsyncTaskExecuter interface {
	ExecTask(task Task) error
}
type AsyncTaskOption struct {
	AsyncMaxWaitTaskNum int
	//最大异步任务go程数量
	MaxAsyncPoolNum int64
	MinAsyncPoolNum int64
	//最大空闲时间
	AsyncPoolIdelTime time.Duration
	//任务最大失败次数
	AsyncTaskMaxFaildedNum uint64
}
type AsyncTaskOperater interface {
	ExecAsyncTask(task Task) error
	StopWait()
}
type AsyncTaskOperate struct {
	taskname string
	sync.WaitGroup

	//异步任务通道
	asyncchan chan Task
	//减少异步任务通道
	closeasyncchan      chan bool
	asyncMaxWaitTaskNum int
	//最大异步任务go程数量
	maxAsyncPoolNum int64
	//最小任务go程数量
	minAsyncPoolNum int64
	//当前异步go程数量
	curAsyncPoolNum int64

	//最大空闲时间
	asyncPoolIdelTime time.Duration
	//任务最大执行次数
	asyncTaskMaxFaildedNum uint64
	//asyncTaskExecuter      AsyncTaskExecuter
	execHandle func(task Task) error
}

func CreateAsyncTaskOperater(taskname string, asyncTaskExecuter AsyncTaskExecuter, option *AsyncTaskOption) AsyncTaskOperater {
	tmp := &AsyncTaskOperate{
		closeasyncchan: make(chan bool),
		taskname:       taskname,
	}
	if option.AsyncMaxWaitTaskNum > 16 {
		tmp.asyncMaxWaitTaskNum = option.AsyncMaxWaitTaskNum
	} else {
		tmp.asyncMaxWaitTaskNum = 16
	}
	tmp.asyncchan = make(chan Task, tmp.asyncMaxWaitTaskNum)

	if option.AsyncPoolIdelTime > 0 {
		tmp.asyncPoolIdelTime = option.AsyncPoolIdelTime
	} else {
		tmp.asyncPoolIdelTime = 5 * time.Second
	}
	if option.AsyncTaskMaxFaildedNum > 0 {
		tmp.asyncTaskMaxFaildedNum = option.AsyncTaskMaxFaildedNum
	} else {
		tmp.asyncTaskMaxFaildedNum = 1
	}
	if option.MinAsyncPoolNum > 0 {
		tmp.minAsyncPoolNum = option.MinAsyncPoolNum
	} else {
		tmp.minAsyncPoolNum = 1
	}
	if option.MaxAsyncPoolNum > 0 {
		tmp.maxAsyncPoolNum = option.MaxAsyncPoolNum
	} else {
		tmp.maxAsyncPoolNum = tmp.minAsyncPoolNum
	}

	tmp.execHandle = asyncTaskExecuter.ExecTask
	tmp.initPool()
	return tmp
}
func (m *AsyncTaskOperate) StopWait() {

}
func (m *AsyncTaskOperate) initPool() {
	Tlog.Debugf("开始初始化go程[%s]：%d,%d", m.taskname, m.minAsyncPoolNum, m.maxAsyncPoolNum)

	m.curAsyncPoolNum = 0
	for i := int64(0); i < m.minAsyncPoolNum; i++ {
		atomic.AddInt64(&m.curAsyncPoolNum, 1)
		go m.addAsync()
	}
	Tlog.Debugf("初始化go程[%v]完成", m.taskname)
	if m.minAsyncPoolNum < m.maxAsyncPoolNum {
		go func() {

			for {
				cur := len(m.asyncchan) * int(m.maxAsyncPoolNum) / int(m.asyncMaxWaitTaskNum)
				if (m.curAsyncPoolNum < m.minAsyncPoolNum || m.curAsyncPoolNum < m.maxAsyncPoolNum) &&
					int(m.curAsyncPoolNum) < cur {
					n := cur - int(m.curAsyncPoolNum)
					if n > 10 {
						n = 10
					}
					for i := 0; i < n; i++ {
						atomic.AddInt64(&m.curAsyncPoolNum, 1)
						go m.addAsync()
					}
					time.Sleep(time.Millisecond)
					continue
				}
				//	Tlog.Infof("任务池[%v] 当前数量:%d (%d-%d)", m.taskname, m.curAsyncPoolNum, m.minAsyncPoolNum, m.maxAsyncPoolNum)
				time.Sleep(time.Second)

			}

		}()
	}

}

func (m *AsyncTaskOperate) ExecAsyncTask(task Task) error {
	m.asyncchan <- task
	// if m.minAsyncPoolNum < m.maxAsyncPoolNum {
	// 	m.cond.L.Lock()
	// 	m.cond.Signal()
	// 	m.cond.L.Unlock()
	// }
	return nil
}

//增加一个异步任务
func (m *AsyncTaskOperate) addAsync() {
	m.Add(1)
	defer func() {
		if err := recover(); err != nil {
			Tlog.Warn("异步任务["+m.taskname+"]异常退出", err)
			Tlog.Warn(string(debug.Stack()))
			atomic.AddInt64(&m.curAsyncPoolNum, -1)
		}

		//Tlog.Debug("异步任务["+m.taskname+"]退出:剩余", m.curAsyncPoolNum)
		m.Done()
	}()
	//Tlog.Debug( "异步任务["+m.taskname+"]增加:当前数量", m.curAsyncPoolNum)
	for {
		select {

		case task := <-m.asyncchan:

			m.execTask(task)
			runtime.Gosched()
		case <-time.After(m.asyncPoolIdelTime):

			if m.curAsyncPoolNum > m.minAsyncPoolNum {
				var poolnum = m.curAsyncPoolNum
				if !atomic.CompareAndSwapInt64(&m.curAsyncPoolNum, poolnum, poolnum-1) {
					continue
				}
				//Tlog.Debug("异步任务[" + m.taskname + "]空闲退出")
				return
			}
		}
	}
}
func (m *AsyncTaskOperate) execTask(task Task) {
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		task.faidedAdd()
	// 		Tlog.Error("异步任务执行失败:["+m.taskname+"]", fmt.Errorf("%s %s", task, err))
	// 		debug.PrintStack()
	// 		e, ok := err.(error)
	// 		if !ok {
	// 			e = errors.New("异步任务执行失败")
	// 		}
	// 		task.Failded(e)
	// 	}
	// }()
	if err := m.execHandle(task); err != nil {
		task.faidedAdd()
		if task.GetFaildedNum() < m.asyncTaskMaxFaildedNum {
			m.execTask(task)
		} else {
			//放弃异步写入任务
			Tlog.Warn("异步任务执行失败:["+m.taskname+"]", err)

			task.Failded(err)
		}

	} else {
		task.Success()
	}

}
