package taskpool

import (
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// LongTimeTaskOperate 该go程池可以用来维持长连接任务，可以控制最大数量
//长执行时间任务go程池配置
type longTimeTaskOperate struct {
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

// CreateLongTimeTaskOperater 创建长执行时间任务go程池
//参数taskname 任务池名称
//参数asyncTaskExecuter 任务执行者，决定任务如何执行
//参数option 任务池配置参数
func CreateLongTimeTaskOperater(taskname string, asyncTaskExecuter AsyncTaskExecuter, option *AsyncTaskOption) AsyncTaskOperater {
	tmp := &longTimeTaskOperate{
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

	//tmp.asyncTaskExecuter = asyncTaskExecuter
	tmp.execHandle = asyncTaskExecuter.ExecTask
	tmp.initPool()
	return tmp
}
func (m *longTimeTaskOperate) SetMaxAsyncPoolNum(num int64) {
	m.maxAsyncPoolNum = num
}
func (m *longTimeTaskOperate) SetMinAsyncPoolNum(num int64) {
	m.minAsyncPoolNum = num
}
func (m *longTimeTaskOperate) SetAsyncPoolIdelTime(idel time.Duration) {
	m.asyncPoolIdelTime = idel
}
func (m *longTimeTaskOperate) StopWait() {

}
func (m *longTimeTaskOperate) initPool() {
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
				for m.curAsyncPoolNum < m.maxAsyncPoolNum && len(m.asyncchan) > 0 {
					atomic.AddInt64(&m.curAsyncPoolNum, 1)
					go m.addAsync()
					time.Sleep(10 * time.Microsecond)
					runtime.Gosched()
					continue
				}
				time.Sleep(time.Second)
			}

		}()
	}

}

func (m *longTimeTaskOperate) ExecAsyncTask(task Task) error {
	m.asyncchan <- task
	// if m.minAsyncPoolNum < m.maxAsyncPoolNum {
	// 	m.cond.L.Lock()
	// 	m.cond.Signal()
	// 	m.cond.L.Unlock()
	// }
	return nil
}

//增加一个异步任务
func (m *longTimeTaskOperate) addAsync() {
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
func (m *longTimeTaskOperate) execTask(task Task) {
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
