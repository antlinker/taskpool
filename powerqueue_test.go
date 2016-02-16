package taskpool_test

import (
	"fmt"
	"math/rand"
	"time"

	. "github.com/antlinker/taskpool"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("测试权重列表功能", func() {
	It("倒序插入", func() {
		list := NewPowerList()
		for i := int64(0); i < 10; i++ {
			for n := int64(0); n < 3; n++ {
				By(fmt.Sprintf("%d:%v", i, n*i))
				list.Insert(i, n*i)
			}
		}
		Ω(list.Len()).Should(Equal(3 * 10))
		for i := int64(9); i >= 0; i -= 1 {
			for n := int64(0); n < 3; n++ {
				elem := list.Pop()

				By(fmt.Sprintf("%d:%v", elem.Lvl(), elem.Get()))
				Ω(elem.Lvl()).Should(Equal(i))
				Ω(elem.Get()).Should(Equal(i * n))
			}
		}

	})
	It("顺序插入", func() {
		list := NewPowerList()
		for i := int64(9); i >= 0; i -= 1 {
			for n := int64(0); n < 3; n++ {
				By(fmt.Sprintf("%d:%v", i, n*i))
				list.Insert(i, n*i)
			}
		}
		Ω(list.Len()).Should(Equal(3 * 10))
		for i := int64(0); i < 10; i++ {
			for n := int64(0); n < 3; n++ {
				elem := list.Pop()

				By(fmt.Sprintf("%d:%v", elem.Lvl(), elem.Get()))
				Ω(elem.Lvl()).Should(Equal(9 - i))
				Ω(elem.Get()).Should(Equal((9 - i) * n))
			}
		}

	})
	It("乱序插入", func() {
		randInt := rand.New(rand.NewSource(time.Now().Unix()))

		list := NewPowerList()
		result := make(map[int]int64)
		for i := 0; i < 100; i++ {
			n := randInt.Int63()
			r := int64(n % 10)
			list.Insert(r, i)
			result[i] = r

		}
		Ω(list.Len()).Should(Equal(100))
		pre := int64(10)
		for i := 0; i < 100; i++ {
			elem := list.Pop()

			By(fmt.Sprintf("%d:%v", elem.Lvl(), elem.Get()))

			Ω(elem.Lvl()).Should(BeNumerically("<=", pre))
			Ω(elem.Lvl()).Should(Equal(result[elem.Get().(int)]))
			pre = elem.Lvl()

		}

	})

	Measure("基准测试插入读取速度", func(b Benchmarker) {
		randInt := rand.New(rand.NewSource(time.Now().Unix()))
		list := NewPowerList()
		//fmt.Printf("有桶%d,桶中有元素数量(%d) 个 \n", tongManage.TongLen(), tongManage.Len())
		runtime := b.Time("runtime", func() {
			for i := 0; i < 10000; i++ {
				n := randInt.Int63()
				r := int64(n % 100)
				list.Insert(r, n)
			}
			//pre := int64(10)
			for i := 0; i < 10000; i++ {

				list.Pop()
				//Ω(elem.Lvl()).Should(BeNumerically("<=", pre))
				//pre = elem.Lvl()
			}
		})
		Ω(runtime.Seconds()).Should(BeNumerically("<", 0.04), "SomethingHard() shouldn't take too long.")
		//b.RecordValue("执行时间:", runtime.Nanoseconds())
	}, 100)
})
