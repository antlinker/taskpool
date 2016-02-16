package taskpool

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

type GoAsyncTaskOperate struct {
	taskname string
	sync.WaitGroup

	//异步任务通道
	asyncchan chan Task

	asyncTaskExecuter   AsyncTaskExecuter
	asyncMaxWaitTaskNum int
	//任务最大执行次数
	asyncTaskMaxFaildedNum uint64
	curWaitTaskNum         int64
	quit                   bool
}

func CreateGoAsyncTaskOperater(taskname string, asyncTaskExecuter AsyncTaskExecuter, option *AsyncTaskOption) AsyncTaskOperater {
	tmp := &GoAsyncTaskOperate{
		taskname: taskname,
	}
	if option.AsyncMaxWaitTaskNum > 16 {
		tmp.asyncMaxWaitTaskNum = option.AsyncMaxWaitTaskNum
	} else {
		tmp.asyncMaxWaitTaskNum = 16
	}
	if option.AsyncTaskMaxFaildedNum > 0 {
		tmp.asyncTaskMaxFaildedNum = option.AsyncTaskMaxFaildedNum
	} else {
		tmp.asyncTaskMaxFaildedNum = 1
	}
	tmp.asyncchan = make(chan Task, tmp.asyncMaxWaitTaskNum)
	tmp.asyncTaskExecuter = asyncTaskExecuter
	tmp.initPool()
	return tmp
}

func (m *GoAsyncTaskOperate) initPool() {
	Tlog.Debugf("开始初始化任务池[%v]%d,%d", m.taskname)
	go m.readTask()
	Tlog.Debugf("初始化任务池[%v]完成", m.taskname)
}
func (m *GoAsyncTaskOperate) StopWait() {
	m.quit = true
	m.Wait()
}
func (m *GoAsyncTaskOperate) readTask() {

	for task := range m.asyncchan {
		m.Add(1)
		go m.execTask(task)
	}
}
func (m *GoAsyncTaskOperate) ExecAsyncTask(task Task) error {
	if m.quit {
		return errors.New("任务池已经停止,不能执行任务")
	}
	m.asyncchan <- task
	return nil
}
func (m *GoAsyncTaskOperate) execTask(task Task) {
	defer func() {
		if err := recover(); err != nil {
			Tlog.Error("异步任务执行失败:["+m.taskname+"]", fmt.Errorf("%s %s", task, err))
			debug.PrintStack()
			e, ok := err.(error)
			if !ok {
				e = errors.New("异步任务执行失败")
				task.Failded(e)
			}
		}
		m.Done()
	}()
	if err := m.asyncTaskExecuter.ExecTask(task); err != nil {
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
