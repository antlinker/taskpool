/*
实现了go程池短任务管理。
保证go程释放后不被回收下次任务可以重复利用,已解决大并发多任务时go程复用。
创建代码如下：

	CreateAsyncTaskOperater(taskname string, asyncTaskExecuter AsyncTaskExecuter, option *AsyncTaskOption) AsyncTaskOperater

实现了go程池长任务管理。
保证go程释放后不被回收下次任务可以重复利用,用来解决长连接大量断开重连时导致的go程的频繁释放，同时可以控制长连接的最大数量。
创建代码如下：

	CreateLongTimeTaskOperater(taskname string, asyncTaskExecuter AsyncTaskExecuter, option *AsyncTaskOption) AsyncTaskOperater

go程池任务执行需要实现AsyncTaskExecuter 接口
go程池任务需要实现Task接口。
go程池配置可以通过AsyncTaskOption 进行设置。
可以控制go程最大最小数量，任务执行失败的次数，重试次数0，或1，都仅能执行一次，大于1则需要多次执行。直到执行次数后仍然失败才返回执行失败。


实现了一个异步任务分任务池，可以根据权重局部进行排序。
使用以下代码创建任务池：

	CreateDefaultTaskPool(option Option) TaskPooler

可以通过option Option控制任务池的配置。
任务池执行的任务需要实现Tasker接口
*/

package taskpool
