/**
任务缓存池，任务可以分片优先级排序
**/
package taskpool

import (
	"fmt"
	"time"
)

type TaskPoolExecError struct {
	err interface{}
}

func (e *TaskPoolExecError) Error() string {
	return fmt.Sprintf("任务执行失败：%v", e.err)
}

//执行任务级别
type TaskLevel int64

const (
	//最低级别任务
	TaskLevel_Lowest TaskLevel = iota * 1024
	//较低级别任务
	TaskLevel_Lower
	//正常级别任务
	TaskLevel_Normal
	//较高级别任务
	TaskLevel_Higher
	//最高级别任务
	TaskLevel_Highest
)

type Option struct {
	//进入任务队列的缓冲大小
	PutBuffNum int
	//任务队列分片内最大任务数
	SliceElemMaxNum int
	//任务队列分片总数
	SliceTotalNum int
	//任务go程数量
	TaskPoolNum int64
}

//任务池
type TaskPooler interface {
	Put(task Tasker) Takener
	Stop()
}
type Tasker interface {
	Lvl() TaskLevel
	Exec() (interface{}, error)
}

//执行任务返回
type Takener interface {
	//等待任务执行完成
	Wait()
	//任务执行结果
	Result() interface{}
	//任务执行是否完成
	//true 完成，false未完成
	IsEnd() bool
	//任务创建时间
	CreTime() time.Time
	//任务完成时间
	EndTime() time.Time
	//是否执行失败，error为nil执行成功，否则为失败，error表示失败原因
	Error() error
}
