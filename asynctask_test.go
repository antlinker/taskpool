package taskpool_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/antlinker/taskpool"

	//	. "github.com/onsi/ginkgo"
	//	. "github.com/onsi/gomega"
)

type GoTestTask struct {
	BaseTask
	id uint64
}

type TestAsyncTaskExecuter struct {
	sync.WaitGroup
	end        chan bool
	maxTaskNum int
	runTaskNum uint64
}

func (e *TestAsyncTaskExecuter) ExecTask(task Task) error {
	atomic.AddUint64(&e.runTaskNum, 1)
	time.Sleep(100 * time.Millisecond)
	e.Done()
	return nil
}

//var _ = Describe("asynctask", func() {

//	var (
//		taskExecuter = &TestAsyncTaskExecuter{runTaskNum: 0, maxTaskNum: 1000000}
//		taskOperate  = CreateAsyncTaskOperater("测试异步ｇｏ程池的效率", taskExecuter, &AsyncTaskOption{
//			AsyncMaxWaitTaskNum: 1024 * 1024,
//			//最大异步任务go程数量
//			MaxAsyncPoolNum: 1024 * 1024 * 1024,
//			MinAsyncPoolNum: 1024,
//			//最大空闲时间
//			AsyncPoolIdelTime: 30 * time.Second,
//			//任务最大失败次数
//			AsyncTaskMaxFaildedNum: 0,
//		})
//	)

//	Measure("基准测试task效率", func(b Benchmarker) {
//		runtime := b.Time("runtime", func() {
//			for i := uint64(0); i < taskExecuter.maxTaskNum; i++ {
//				//atomic.AddUint64(&taskExecuter.maxTaskNum, 1)
//				taskExecuter.Add(1)
//				taskOperate.ExecAsyncTask(&GoTestTask{})
//			}
//		})

//		Ω(runtime.Seconds()).Should(BeNumerically("<", 2), "SomethingHard() shouldn't take too long.")
//		//taskExecuter.Wait()
//		//b.RecordValue("执行时间:", runtime.Seconds())
//	}, 100)
//})
var (
	n      = 1000
	s      = 1024
	maxNum = int64(40960)
	minNum = int64(10240)
)

func TestAAA(t *testing.T) {

	taskExecuter := &TestAsyncTaskExecuter{runTaskNum: 0}
	taskOperate := CreateAsyncTaskOperater("测试异步ｇｏ程池的效率", taskExecuter, &AsyncTaskOption{
		AsyncMaxWaitTaskNum: 1024,
		//最大异步任务go程数量
		MaxAsyncPoolNum: maxNum,
		MinAsyncPoolNum: minNum,
		//最大空闲时间
		AsyncPoolIdelTime: 3 * time.Second,
		//任务最大失败次数
		AsyncTaskMaxFaildedNum: 0,
	})
	time.Sleep(3 * time.Second)
	starttime := time.Now()
	taskExecuter.maxTaskNum = n * s
	for i := 0; i < n; i++ {
		//taskExecuter.maxTaskNum = 1024000
		taskExecuter.Add(1)
		go func() {
			for i := 0; i < s; i++ {
				//atomic.AddUint64(&taskExecuter.maxTaskNum, 1)
				taskExecuter.Add(1)
				taskOperate.ExecAsyncTask(&GoTestTask{})
			}
			taskExecuter.Done()
		}()

	}

	taskExecuter.Wait()
	endtime := time.Now()
	totaltime := endtime.Sub(starttime)

	t.Log("starttime:", starttime)
	t.Log("endtime:", endtime)
	t.Log("totaltime:", totaltime)
	t.Log("完成任务：", taskExecuter.runTaskNum, "总：", totaltime, "平均：", int(totaltime)/int(taskExecuter.maxTaskNum), "纳秒")
}
