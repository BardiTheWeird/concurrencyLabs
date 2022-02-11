package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

var (
	formatLog func(string, ...interface{}) = func(format string, a ...interface{}) {
		fmt.Printf(format, a...)
	}
)

type Statistics struct {
	TotalProcesses     int
	FinishedProcesses  int
	LostProcesses      int
	DestroyedProcesses int
	MaxQueueLength     int

	curQueueLength int
	mutex          sync.Mutex
}

func (s *Statistics) ModifyStatistics(f func(*Statistics)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	f(s)
}

func (s *Statistics) ChangeQueueLength(amount int) {
	s.ModifyStatistics(func(s *Statistics) {
		s.curQueueLength += amount
		if s.curQueueLength > s.MaxQueueLength {
			s.MaxQueueLength = s.curQueueLength
		}
	})
}

func (s *Statistics) Print() {
	fmt.Println("Processes:")
	fmt.Printf("  Total     %d\n", s.TotalProcesses)
	fmt.Printf("  Finished  %d\t%f%%\n", s.FinishedProcesses, float32(s.FinishedProcesses)/float32(s.TotalProcesses)*100)
	fmt.Printf("  Lost      %d\t%f%%\n", s.LostProcesses, float32(s.LostProcesses)/float32(s.TotalProcesses)*100)
	fmt.Printf("  Destroyed %d\t%f%%\n", s.DestroyedProcesses, float32(s.DestroyedProcesses)/float32(s.TotalProcesses)*100)
	fmt.Println("Max queue length:", s.MaxQueueLength)
}

type Process struct {
	ParentId int
	Id       int
}

type ProcessGenerator struct {
	Id                  int
	ProcessesToGenerate int
	SchedulerQueue      chan Process

	MinProcessGenerationTime int
	MaxProcessGenerationTime int

	Wg *sync.WaitGroup
}

func (p *ProcessGenerator) Run() {
	randSource := rand.NewSource(time.Now().UnixNano())
	rand := *rand.New(randSource)

	for i := 0; i < p.ProcessesToGenerate; i++ {
		// Simulate activity
		timeToWait := (rand.Int() % (p.MaxProcessGenerationTime - p.MinProcessGenerationTime)) + p.MinProcessGenerationTime
		time.Sleep(time.Millisecond * time.Duration(timeToWait))

		process := Process{
			ParentId: p.Id,
			Id:       i,
		}
		formatLog("GEN%d: ==> %d_%d\n", process.ParentId, process.ParentId, process.Id)

		p.SchedulerQueue <- process
	}

	p.Wg.Done()
}

type Cpu struct {
	Id           int
	GeneralQueue chan Process
	DirectQueue  chan Process
	Wg           *sync.WaitGroup

	*Statistics

	IsBusy          bool
	CurProcess      Process
	CurProcessMutex sync.Mutex

	MinProcessingTime int
	MaxProcessingTime int
}

func (c *Cpu) Run() {
	randSource := rand.NewSource(time.Now().UnixNano())
	rand := *rand.New(randSource)

	runProcess := func(p Process) {
		c.CurProcessMutex.Lock()
		c.IsBusy = true
		c.CurProcess = p
		c.CurProcessMutex.Unlock()

		formatLog("CPU%d: <== %d_%d\n", c.Id, p.ParentId, p.Id)

		// simulate activity
		timeToWait := (rand.Int() % (c.MaxProcessingTime - c.MinProcessingTime)) + c.MinProcessingTime
		time.Sleep(time.Millisecond * time.Duration(timeToWait))

		formatLog("CPU%d: finished %d_%d\n", c.Id, p.ParentId, p.Id)

		c.CurProcessMutex.Lock()
		c.IsBusy = false
		c.CurProcessMutex.Unlock()
	}

	var process Process
	for {
		select {
		case p, ok := <-c.GeneralQueue:
			if !ok {
				c.Wg.Done()
				return
			}
			process = p
			c.ChangeQueueLength(-1)
		case process = <-c.DirectQueue:
		}

		runProcess(process)
	}
}

func (c *Cpu) GetCurrentProcess() (Process, bool) {
	c.CurProcessMutex.Lock()
	defer c.CurProcessMutex.Unlock()
	return c.CurProcess, c.IsBusy
}

type Scheduler struct {
	Gen1  *ProcessGenerator
	Gen2  *ProcessGenerator
	GenWg *sync.WaitGroup

	Cpu1 *Cpu
	Cpu2 *Cpu

	*Statistics

	SchedulerQueue chan Process
	CpuQueue       chan Process
}

func (s *Scheduler) Run() {
	noMoreProccessesChannel := make(chan bool)
	go func() {
		s.GenWg.Wait()
		noMoreProccessesChannel <- true
	}()

	pushToDirectQueue := func(c *Cpu, p Process) {
		formatLog("SCHD: %d_%d ==> Cpu%d\n", p.ParentId, p.Id, c.Id)
		c.DirectQueue <- p
	}

	scheduleProcess := func(p Process) {
		switch p.ParentId {
		case 1:
			cpu1Process, cpu1Busy := s.Cpu1.GetCurrentProcess()
			if !cpu1Busy {
				pushToDirectQueue(s.Cpu1, p)
				return
			}
			if cpu1Process.ParentId != 1 {
				formatLog("SCHD: %d_%d is lost\n", p.ParentId, p.Id)
				s.ModifyStatistics(func(s *Statistics) { s.LostProcesses++ })
				return
			}
			_, cpu2Busy := s.Cpu2.GetCurrentProcess()
			if !cpu2Busy {
				pushToDirectQueue(s.Cpu2, p)
				return
			}
			formatLog("SCHD: %d_%d is destroyed\n", p.ParentId, p.Id)
			s.ModifyStatistics(func(s *Statistics) { s.DestroyedProcesses++ })
		case 2:
			_, cpu2Busy := s.Cpu2.GetCurrentProcess()
			if cpu2Busy {
				s.CpuQueue <- p
				s.ChangeQueueLength(1)
				formatLog("SCHD: %d_%d ==> CpuQueue\n", p.ParentId, p.Id)
			} else {
				pushToDirectQueue(s.Cpu2, p)
			}
		default:
			formatLog("Process has invalid ParentId = %d\n", p.ParentId)
		}
	}

	for {
		select {
		case p := <-s.SchedulerQueue:
			scheduleProcess(p)
		case <-noMoreProccessesChannel:
			close(s.CpuQueue)
			return
		}
	}
}

func main() {
	g1p := flag.Int("g1p", 15, "number of processes for GEN1 to generate")
	g1m := flag.Int("g1m", 50, "min GEN1 process generation time")
	g1M := flag.Int("g1M", 300, "max GEN1 process generation time")

	g2p := flag.Int("g2p", 15, "number of processes for GEN1 to generate")
	g2m := flag.Int("g2m", 50, "min GEN1 process generation time")
	g2M := flag.Int("g2M", 300, "max GEN1 process generation time")

	c1m := flag.Int("c1m", 60, "min CPU1 processing time")
	c1M := flag.Int("c1M", 200, "max CPU1 processing time")

	c2m := flag.Int("c2m", 30, "min CPU1 processing time")
	c2M := flag.Int("c2M", 100, "max CPU1 processing time")

	logOn := flag.Bool("log", false, "whether to log runtime info")
	printHelp := flag.Bool("help", false, "print this message")

	flag.Parse()

	if *printHelp {
		flag.Usage()
		os.Exit(0)
	}

	if !*logOn {
		formatLog = func(s string, i ...interface{}) {}
	}

	stat := Statistics{
		TotalProcesses: *g1p + *g2p,
	}

	schedulerQueue := make(chan Process)
	cpuQueue := make(chan Process, *g1p+*g2p)
	var genWg sync.WaitGroup
	genWg.Add(2)
	var cpuWg sync.WaitGroup
	cpuWg.Add(2)

	gen1 := ProcessGenerator{
		Id:             1,
		SchedulerQueue: schedulerQueue,
		Wg:             &genWg,

		ProcessesToGenerate:      *g1p,
		MinProcessGenerationTime: *g1m,
		MaxProcessGenerationTime: *g1M,
	}

	gen2 := ProcessGenerator{
		Id:             2,
		SchedulerQueue: schedulerQueue,
		Wg:             &genWg,

		ProcessesToGenerate:      *g2p,
		MinProcessGenerationTime: *g2m,
		MaxProcessGenerationTime: *g2M,
	}

	cpu1 := Cpu{
		Id:           1,
		GeneralQueue: cpuQueue,
		DirectQueue:  make(chan Process),
		Wg:           &cpuWg,

		Statistics: &stat,

		MinProcessingTime: *c1m,
		MaxProcessingTime: *c1M,
	}

	cpu2 := Cpu{
		Id:           2,
		GeneralQueue: cpuQueue,
		DirectQueue:  make(chan Process),
		Wg:           &cpuWg,

		Statistics: &stat,

		MinProcessingTime: *c2m,
		MaxProcessingTime: *c2M,
	}

	scheduler := Scheduler{
		Gen1: &gen1,
		Gen2: &gen2,
		Cpu1: &cpu1,
		Cpu2: &cpu2,

		Statistics: &stat,

		SchedulerQueue: schedulerQueue,
		CpuQueue:       cpuQueue,
		GenWg:          &genWg,
	}

	go cpu1.Run()
	go cpu2.Run()
	go gen1.Run()
	go gen2.Run()

	fmt.Println("Running...")

	scheduler.Run()
	cpuWg.Wait()

	stat.FinishedProcesses = stat.TotalProcesses - stat.LostProcesses - stat.DestroyedProcesses
	fmt.Println()
	stat.Print()
}
