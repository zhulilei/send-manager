package main

import (
	"fmt"
	"time"

	"github.com/astaxie/beego"
)

const (
	// STATUS_ORIGINAL 代表原始。
	STATUS_ORIGINAL uint32 = 0
	// STATUS_STARTING 代表正在启动。
	STATUS_STARTING uint32 = 1
	// STATUS_STARTED 代表已启动。
	STATUS_STARTED uint32 = 2
	// STATUS_STOPPING 代表正在停止。
	STATUS_STOPPING uint32 = 3
	// STATUS_STOPPED 代表已停止。
	STATUS_STOPPED uint32 = 4
)

const closed = "closed"

type T struct {
	timeout      time.Duration // 处理超时时间，单位：纳秒。
	concurrency  uint32        // 载荷并发量。
	tickets      GoTickets     // Goroutine票池
	quit         chan struct{}
	status       uint32
	pause        chan struct{} //停顿chan
	pauseTimeout time.Duration //停顿的时间
	// wg          sync.WaitGroup
}

func NewT() (t *T, err error) {
	t = &T{
		timeout:      20 * time.Second,
		concurrency:  10,
		quit:         make(chan struct{}),
		pause:        make(chan struct{}, 1),
		pauseTimeout: 5 * time.Second,
		// wg:          sync.WaitGroup{},
	}
	// t.pause = make(chan struct{}, t.concurrency)
	if err := t.init(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *T) init() error {
	//初始化票池
	tickets, err := NewGoTickets(t.concurrency)
	if err != nil {
		return err
	}
	t.tickets = tickets
	return nil
}

func (t *T) Start() {
	fmt.Println("Starting T")
	go func() {
		for {
			select {
			case <-t.pause:
				fmt.Println("get pause signal so sleep .........")
				time.Sleep(t.pauseTimeout)
				fmt.Println("finish pause signal so sleep .........")
			case <-t.quit:
				beego.Error("closed")
				return
			default:
				t.G()
			}
		}
	}()
	t.status = STATUS_STARTED
}

//接受signals信号的时候close
func (t *T) Close() {
	close(t.quit)
	// t.wg.Wait()
	for {
		if t.tickets.Remainder() == t.concurrency {
			break
		}
	}
	close(t.pause)
	beego.Error("close successful")
}

func (t *T) G() {
	var gId int
	gId = getRand()

	select {
	case <-t.quit:
		beego.Error("G get quit signal")
		return
	default:
		t.tickets.Take()
		// t.wg.Add(1)
		fmt.Println("this is goroutine ", gId)
		go func() {
			var isSuccess bool //true->update ticket success;false -> update ticket false

			defer func() {
				beego.Error(gId, " t.tickets.Return()")
				t.tickets.Return()
			}()

			// defer func() {
			// 	beego.Error(gId, " t.wg.Done()")
			// 	t.wg.Done()
			// }()

			defer func() {
				//需要考虑意外停止的时候
				if err := recover(); err != nil {
					time.Sleep(1 * time.Second)
					beego.Error("panic")
					beego.Error(gId, "update ticket idle")
					return
				}
				if isSuccess {
					beego.Error(gId, "update ticket success")
				} else {
					beego.Error(gId, "update ticket idle")
				}
			}()

			timeout := time.After(t.timeout)
			select {
			case result := <-t.generateResult(gId):
				beego.Error(gId, " gId get result", result)
				if result {
					isSuccess = true
					return
				}
				isSuccess = false //这里需要通知main goroutine 停顿一段时间再起g
				return
			case <-timeout:
				beego.Error(gId, " timeout")
				isSuccess = false
				return
			case <-t.quit:
				beego.Error(gId, " receive close signal")
				isSuccess = false
				// panic(closed)
				return
			}

			//if result=false -> update ticket=idle
			//if result=true  -> update ticket=success
			//if timeout -> update ticket=idle
		}()
	}

}

//result:
// - true 都发送成功
// - false 发送有失败或者需要重试的
// - not found // -len(items)为0,这时候需要main goroutine休息一段时间再起
func (t *T) generateResult(gId int) <-chan bool {
	cc := make(chan bool, 1)
	items := getItems() //去中间表取shard find And update status=busy
	fmt.Println(gId, "items ", items)
	go func() {
		var result bool
		result = true
		if len(items) > 0 {
			status := make(chan bool, len(items))
			for _, item := range items {
				go t.do(gId, item, status)
			}
			for i := 0; i < len(items); i++ {
				c := <-status
				if !c {
					result = false
				}
			}
			fmt.Printf("%d gid result:%v\n", gId, result)
			cc <- result
		} else {
			select {
			case <-t.pause:
			default:
				fmt.Println("start send pause sig")
				t.pause <- struct{}{}
				fmt.Println("finish send pause sig")
			}
			result = false
			cc <- false
		}
	}()
	return cc
}

// //发送并且更新message mgo
// func (t *T) do(gId, id int, status chan bool) {
// 	timeSleep := getRand()
// 	fmt.Printf("gId:%d-taskId:%d-sleep %d s\n", gId, id, timeSleep)
// 	time.Sleep(time.Duration(timeSleep) * time.Second)
// 	fmt.Printf("gId:%d-taskId:%d-do\n", gId, id)
// 	err := Mail()
// 	if err != nil {
// 		status <- false
// 		fmt.Printf("gId:%d-taskId:%d-update message mgo---send  mail unsuccessfully\n", gId, id)
// 		return
// 	}
// 	fmt.Printf("gId:%d-taskId:%d-update message mgo---send  mail successfully\n", gId, id)

// 	err = SMS()
// 	if err != nil {
// 		status <- false
// 		fmt.Printf("gId:%d-taskId:%d-update message mgo---send sms unsuccessfully\n", gId, id)
// 		return
// 	}
// 	fmt.Printf("gId:%d-taskId:%d-update message mgo---send  sms successfully\n", gId, id)

// 	status <- true
// }

//发送并且更新message mgo
func (t *T) do(gId, id int, status chan bool) {
	timeSleep := getRand()
	fmt.Printf("gId:%d-taskId:%d-sleep %d s\n", gId, id, timeSleep)
	time.Sleep(time.Duration(timeSleep) * time.Second)
	fmt.Printf("gId:%d-taskId:%d-do\n", gId, id)
	status <- true
}
