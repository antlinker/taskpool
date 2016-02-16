package taskpool

import (
	"errors"
	"sync"
	"time"
)

var PoolStopError = errors.New("任务池已经停止")

//任务
type pooltask struct {
	BaseTask
	task  Tasker
	taken *Taken
}

var takenPool = sync.Pool{
	New: func() interface{} {
		return &Taken{}
	},
}

type Taken struct {
	resultChan chan struct{}
	err        error
	result     interface{}
	startTime  time.Time
	endTime    time.Time
	end        bool
}

func (t *Taken) Wait() {

	<-t.resultChan

}
func (t *Taken) start() {
	t.end = false
	t.err = nil
	t.startTime = time.Now()
	t.resultChan = make(chan struct{})
}
func (t *Taken) end0(result interface{}, err error) {
	t.result = result
	t.err = err
	t.end = true
	close(t.resultChan)
}
func (t *Taken) Result() interface{} {
	return t.result
}

//任务执行是否完成
//true 完成，false未完成
func (t *Taken) IsEnd() bool {
	return t.end
} //任务创建时间

func (t *Taken) CreTime() time.Time {
	return t.startTime
}

//任务完成时间
func (t *Taken) EndTime() time.Time {
	return t.endTime
}

//是否执行失败，error为nil执行成功，否则为失败，error表示失败原因

func (t *Taken) Error() error {
	return t.err
}

//创建默认任务池
//putbuffnum 进入任务缓冲大小
//slicemaxnum任务队列分片内最大任务数

func CreateDefaultTaskPool(option Option) TaskPooler {
	t := &TaskPool{
		option: option,
	}
	t.init()
	return t
}

type TaskPool struct {
	//配置参数
	option Option
	//任务进入通道
	putchan             chan *pooltask
	run                 bool
	queue               SliceQueuer
	taskOperater        AsyncTaskOperater
	highestTaskOperater AsyncTaskOperater
	sync.WaitGroup
}

func (p *TaskPool) init() {
	if p.option.PutBuffNum < 16 {
		p.option.PutBuffNum = 16
	}
	p.run = true
	p.putchan = make(chan *pooltask, p.option.PutBuffNum)
	p.queue = CreateSliceQueue(p.option.SliceTotalNum, p.option.SliceElemMaxNum)

	p.taskOperater = CreateAsyncTaskOperater("基础任务池", p, &AsyncTaskOption{
		AsyncMaxWaitTaskNum: int(p.option.TaskPoolNum),
		//最大异步任务go程数量
		MaxAsyncPoolNum: p.option.TaskPoolNum,
		//MinAsyncPoolNum: p.option.TaskPoolNum / 10,
		MinAsyncPoolNum: 1,

		//任务最大失败次数
		AsyncTaskMaxFaildedNum: 0,
	})
	// p.highestTaskOperater = CreateAsyncTaskOperater("最高优先级任务池", p, &AsyncTaskOption{
	// 	AsyncMaxWaitTaskNum: int(p.option.TaskPoolNum / 2),
	// 	//最大异步任务go程数量
	// 	MaxAsyncPoolNum: p.option.TaskPoolNum / 2,
	// 	MinAsyncPoolNum: p.option.TaskPoolNum / 4,

	// 	//任务最大失败次数
	// 	AsyncTaskMaxFaildedNum: 0,
	// })
	go p.putstart()
	go p.readstart()
}

func (p *TaskPool) Put(task Tasker) Takener {

	ptask := p.createTask(task)
	if !p.run {
		ptask.taken.err = PoolStopError
		return ptask.taken
	}
	p.putchan <- &ptask
	return ptask.taken
}
func (p *TaskPool) Stop() {
	Tlog.Debug("停止分片权重优先排序任务池....")
	//fmt.Println("停止TaskPool")
	p.run = false
	time.Sleep(10 * time.Millisecond)
	close(p.putchan)
	p.queue.Close()
	p.Wait()
	Tlog.Debug("停止基础任务池....")
	p.taskOperater.StopWait()
	Tlog.Debug("停止基础任务池完成")
	Tlog.Debug("停止分片权重优先排序任务池完成")
	//fmt.Println("停止TaskPool完成")
}
func (p *TaskPool) ExecTask(t Task) error {
	task := t.(*pooltask)
	taken := task.taken
	defer func() {
		if err := recover(); err != nil {
			taken.err = &TaskPoolExecError{err}
		}
		taken.endTime = time.Now()
		taken.end = true
		//fmt.Println("执行任务完成退出")
		p.Done()
	}()
	result, err := task.task.Exec()
	//fmt.Println("执行任务完成1")
	taken.end0(result, err)
	//fmt.Println("执行任务完成2")
	return err
}
func (p *TaskPool) createTask(task Tasker) pooltask {
	taken := &Taken{}
	taken.start()
	return pooltask{
		task: task, taken: taken,
	}
}

func (p *TaskPool) putstart() {
	defer func() {
		Tlog.Debug("加入任务队列go程关闭")
	}()
	p.Add(1)
	defer p.Done()
	for {
		select {
		case task := <-p.putchan:
			if task == nil {
				return
			}
			p.Add(1)
			//fmt.Println("获取到一个任务加入到队列")
			// if task.task.Lvl() >= TaskLevel_Highest {
			// 	p.highestTaskOperater.ExecAsyncTask(task)
			// 	continue
			// }
			p.queue.Put(int64(task.task.Lvl()), task)

		case <-time.After(time.Second):
		}
	}
	//fmt.Println("任务准备go程退出")
}
func (p *TaskPool) readstart() {
	defer func() {
		Tlog.Debug("读取任务队列go程关闭")
	}()
	p.Add(1)
	defer p.Done()
	for {
		t := p.queue.PopBlock()
		if t == nil {

			//	fmt.Println("读取到一个空任务退出")
			break
		}

		task := t.Get().(*pooltask)
		//fmt.Println("读取到一个任务，放入任务池执行:", task.lvl)
		p.taskOperater.ExecAsyncTask(task)
	}
	//fmt.Println("任务读取go程退出")
}
