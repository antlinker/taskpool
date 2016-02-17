package taskpool_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/antlinker/taskpool"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type TestTask struct {
	lvl      TaskLevel
	value    interface{}
	result   interface{}
	callback func(*TestTask)
}

func createTestTask(lvl TaskLevel, value interface{}, result interface{}) *TestTask {
	return &TestTask{lvl, value, result, nil}

}

func (t *TestTask) Exec() (result interface{}, err error) {
	//fmt.Println("执行任务TestTask")
	time.Sleep(10 * time.Millisecond)
	if t.callback != nil {
		t.callback(t)
	}
	return t, nil
}
func (t *TestTask) Lvl() TaskLevel {
	return t.lvl
}

var _ = Describe("任务池测试", func() {
	var (
		taskpool TaskPooler
	)
	var _ = BeforeSuite(func() {
		taskpool = CreateDefaultTaskPool(Option{
			//进入任务队列的缓冲大小
			PutBuffNum: 1024,
			//任务队列分片内最大任务数
			SliceElemMaxNum: 1024,
			//任务队列分片总数
			SliceTotalNum: 128,
			//任务go程最大数量
			TaskPoolNum: 102400,
		})
	})
	var _ = AfterSuite(func() {
		taskpool.Stop()
	})
	It("同步任务测试1", func() {
		for i := int64(0); i < 100; i++ {
			task := createTestTask(TaskLevel_Normal, i, 100)
			token := taskpool.Put(task)
			token.Wait()
			if token.IsEnd() {
				Ω(token.Error()).NotTo(HaveOccurred())
				Ω(token.Result()).Should(Equal(task))

			} else {
				Ω(false).Should(BeTrue())
			}
		}
	})
	It("同步任务测试2,测试token Wait", func() {
		task := createTestTask(TaskLevel_Normal, 1, 2)
		token := taskpool.Put(task)
		time.Sleep(time.Second)
		token.Wait()
		if token.IsEnd() {
			Ω(token.Error()).NotTo(HaveOccurred())
			Ω(token.Result()).Should(Equal(task))

		} else {
			Ω(false).Should(BeTrue())
		}

	})
	It("异步任务测试,测试", func() {
		tokens := make(chan Takener, 10000)
		count := int64(0)
		for i := int64(0); i < 100; i++ {
			go func() {
				for lvl := TaskLevel(0); lvl < 10; lvl++ {
					atomic.AddInt64(&count, 1)
					task := createTestTask(lvl, fmt.Sprintf("%v", count), lvl)
					task.callback = func(tt *TestTask) {
						//fmt.Println("任务序号:", tt.value, "任务级别", tt.result)
						Ω(task).Should(Equal(tt))
					}
					tokens <- taskpool.Put(task)
				}
			}()
		}
		var k = 0
		for token := range tokens {
			token.Wait()
			k++
			if k >= 1000 {
				break
			}
		}
	})
	Measure("测试一组1000个任务执行完成时间", func(b Benchmarker) {

		//fmt.Printf("有桶%d,桶中有元素数量(%d) 个 \n", tongManage.TongLen(), tongManage.Len())

		runtime := b.Time("runtime", func() {
			wg := sync.WaitGroup{}
			count := int64(0)
			for i := int64(0); i < 100; i++ {
				wg.Add(1)
				go func() {

					for lvl := TaskLevel(0); lvl < 10; lvl++ {
						atomic.AddInt64(&count, 1)
						task := createTestTask(lvl, fmt.Sprintf("%v", count), lvl)
						task.callback = func(tt *TestTask) {
							//fmt.Println("任务序号:", tt.value, "任务级别", tt.result)
							wg.Done()
							Ω(task).Should(Equal(tt))
						}
						wg.Add(1)
						_ = taskpool.Put(task)
					}
					wg.Done()
				}()
			}
			wg.Wait()
		})
		Ω(runtime.Seconds()).Should(BeNumerically("<", 4), "SomethingHard() shouldn't token too long.")
		b.RecordValue("执行时间:", float64(runtime.Nanoseconds())/1000)
	}, 1000)
	Measure("测试一组1000个任务直接执行完成时间", func(b Benchmarker) {

		//fmt.Printf("有桶%d,桶中有元素数量(%d) 个 \n", tongManage.TongLen(), tongManage.Len())
		runtime := b.Time("runtime", func() {
			wg := sync.WaitGroup{}
			count := int64(0)
			for i := int64(0); i < 100; i++ {

				for lvl := TaskLevel(0); lvl < 10; lvl++ {
					atomic.AddInt64(&count, 1)
					task := createTestTask(lvl, fmt.Sprintf("%v", count), lvl)
					task.callback = func(tt *TestTask) {
						//fmt.Println("任务序号:", tt.value, "任务级别", tt.result)
						wg.Done()
						Ω(task).Should(Equal(tt))
					}
					wg.Add(1)
					task.Exec()
				}

			}
			wg.Wait()

		})
		Ω(runtime.Seconds()).Should(BeNumerically("<", 4), "SomethingHard() shouldn't token too long.")

		b.RecordValue("执行时间:", float64(runtime.Nanoseconds())/1000)
	}, 1000)
})
