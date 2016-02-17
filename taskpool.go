package taskpool

import (
	"errors"
	"fmt"
	"time"
)

//任务池已经停止错误
//放入元素时如果任务池已停止会在tokener中返回该错误
var PoolStopError = errors.New("任务池已经停止")

//任务池执行失败错误
type TaskPoolExecError struct {
	err interface{}
}

//执行错误
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

//分片有限任务池配置参数
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

//异步任务池
type TaskPooler interface {
	//将任务放入池中
	//返回一个Tokener令牌，可以查看任务执行情况
	Put(task Tasker) Tokener
	//停止任务池
	Stop()
}

//执行任务
type Tasker interface {
	//执行任务级别
	Lvl() TaskLevel
	//执行任务方法
	Exec() (interface{}, error)
}

//执行任务令牌
//可以查询任务执行情况
type Tokener interface {
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
