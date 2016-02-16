package taskpool_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/antlinker/go-cmap"
	. "github.com/antlinker/taskpool"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("测试分片列表分片内部权重排序容器功能", func() {
	It("倒序插入弹出", func() {
		queue := CreateSliceQueue(128, 128)

		for i := int64(0); i < 10; i++ {
			for n := int64(0); n < 1000; n++ {
				By(fmt.Sprintf("%d:%v", i, n*i))
				queue.Put(i, n)
			}
		}
		Ω(queue.Len()).Should(Equal(10000))
		count := 0
		for i := 0; i < 128; i++ {
			pre := int64(10)
			for n := 0; n < 128; n++ {
				elem := queue.Pop()
				if elem == nil {
					Ω(count).Should(Equal(10000))
					break
				}

				count++
				Ω(queue.Len() + count).Should(Equal(10000))
				By(fmt.Sprintf("%d:%v", elem.Lvl(), elem.Get()))
				//Ω(elem.Lvl()).Should(Equal(i))
				Ω(elem.Lvl()).Should(BeNumerically("<=", pre))

			}
		}
	})
	It("乱序插入", func() {
		randInt := rand.New(rand.NewSource(time.Now().Unix()))
		queue := CreateSliceQueue(128, 128)
		result := make(map[int]int64)
		for i := 0; i < 10000; i++ {
			n := randInt.Int63()
			r := int64(n % 10)
			queue.Put(r, i)
			result[i] = r
		}
		Ω(queue.Len()).Should(Equal(10000))
		count := 0
		for i := 0; i < 128; i++ {
			pre := int64(10)
			for n := 0; n < 128; n++ {
				if queue.Len() == 0 {
					Ω(count).Should(Equal(10000))
					break
				}
				elem := queue.PopBlock()
				count++
				Ω(queue.Len() + count).Should(Equal(10000))
				By(fmt.Sprintf("%d:%v", elem.Lvl(), elem.Get()))
				//Ω(elem.Lvl()).Should(Equal(i))
				Ω(elem.Lvl()).Should(BeNumerically("<=", pre))
				Ω(elem.Lvl()).Should(Equal(result[elem.Get().(int)]))
			}
		}
	})
	It("异步断续写入弹出测试", func() {
		randInt := rand.New(rand.NewSource(time.Now().Unix()))
		queue := CreateSliceQueue(128, 128)
		result := cmap.NewConcurrencyMap()
		//result := make(map[int]int64, 1000)
		sendchan := make(chan struct{})
		readchan := make(chan struct{})

		go func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Println(err)
					debug.PrintStack()
				}
			}()
			count := 0
			for {
				pre := int64(10)
				if count == 1000 {
					break
				}
				elem := queue.PopBlock()
				// if elem == nil {
				// 	fmt.Println("空了休眠:", count)
				// 	time.Sleep(100 * time.Millisecond)
				// 	continue
				// }
				count++
				By(fmt.Sprintf("%d:%v", elem.Lvl(), elem.Get()))
				//Ω(elem.Lvl()).Should(Equal(i))
				Ω(elem.Lvl()).Should(BeNumerically("<=", pre))
				r, _ := result.Get(int64(elem.Get().(int)))
				//r := result[elem.Get().(int)]
				//Ω(err).NotTo(HaveOccurred())
				Ω(elem.Lvl()).Should(Equal(r))
				runtime.Gosched()
			}
			close(readchan)
		}()
		go func() {
			for i := 0; i < 1000; i++ {
				n := randInt.Int63()
				r := int64(n % 10)
				queue.Put(r, i)
				result.Set(int64(i), r)
				//result[i] = r
				runtime.Gosched()
			}

			close(sendchan)
		}()
		<-sendchan
		<-readchan
		Ω(queue.Len()).Should(Equal(0))
	})
	Measure("基准测试插入读取速度", func(b Benchmarker) {
		randInt := rand.New(rand.NewSource(time.Now().Unix()))
		queue := CreateSliceQueue(128, 128)
		//fmt.Printf("有桶%d,桶中有元素数量(%d) 个 \n", tongManage.TongLen(), tongManage.Len())
		runtime := b.Time("runtime", func() {
			for i := 0; i < 10000; i++ {
				n := randInt.Int63()
				r := int64(n % 10)
				queue.Put(r, n)
			}
			//pre := int64(10)
			for i := 0; i < 10000; i++ {

				queue.Pop()
				//Ω(elem.Lvl()).Should(BeNumerically("<=", pre))
				//pre = elem.Lvl()
			}
		})
		Ω(runtime.Seconds()).Should(BeNumerically("<", 0.4), "SomethingHard() shouldn't take too long.")
		//b.RecordValue("执行时间:", runtime.Nanoseconds())
	}, 100)
})
