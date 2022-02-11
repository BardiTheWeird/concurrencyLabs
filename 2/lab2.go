package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

type Process struct {
	ParentId int
	Id       int
}

// I need to figure out how to send the DONE signal from here
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
		fmt.Printf("GEN%d: ==> %d_%d\n", process.ParentId, process.ParentId, process.Id)

		p.SchedulerQueue <- process
	}

	p.Wg.Done()
}

type Cpu struct {
	Id           int
	GeneralQueue chan Process
	DirectQueue  chan Process
	Wg           *sync.WaitGroup

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

		fmt.Printf("CPU%d: <== %d_%d\n", c.Id, p.ParentId, p.Id)

		// simulate activity
		timeToWait := (rand.Int() % (c.MaxProcessingTime - c.MinProcessingTime)) + c.MinProcessingTime
		time.Sleep(time.Millisecond * time.Duration(timeToWait))

		fmt.Printf("CPU%d: finished %d_%d\n", c.Id, p.ParentId, p.Id)

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
		fmt.Printf("SCHD: %d_%d ==> Cpu%d\n", p.ParentId, p.Id, c.Id)
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
				fmt.Printf("SCHD: %d_%d is lost\n", p.ParentId, p.Id)
				return
			}
			_, cpu2Busy := s.Cpu2.GetCurrentProcess()
			if !cpu2Busy {
				pushToDirectQueue(s.Cpu2, p)
				return
			}
			fmt.Printf("SCHD: %d_%d is destroyed\n", p.ParentId, p.Id)
		case 2:
			_, cpu2Busy := s.Cpu2.GetCurrentProcess()
			if cpu2Busy {
				s.CpuQueue <- p
				fmt.Printf("SCHD: %d_%d ==> CpuQueue\n", p.ParentId, p.Id)
			} else {
				pushToDirectQueue(s.Cpu2, p)
			}
		default:
			fmt.Println("Process has invalid ParentId =", p.ParentId)
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

	schedulerQueue := make(chan Process)
	cpuQueue := make(chan Process)
	var genWg sync.WaitGroup
	genWg.Add(2)
	var cpuWg sync.WaitGroup
	cpuWg.Add(2)

	gen1 := ProcessGenerator{
		Id:             1,
		SchedulerQueue: schedulerQueue,
		Wg:             &genWg,
	}

	gen2 := ProcessGenerator{
		Id:             2,
		SchedulerQueue: schedulerQueue,
		Wg:             &genWg,
	}

	cpu1 := Cpu{
		Id:           1,
		GeneralQueue: cpuQueue,
		DirectQueue:  make(chan Process),
		Wg:           &cpuWg,

		IsBusy:          false,
		CurProcess:      Process{},
		CurProcessMutex: sync.Mutex{},
	}

	cpu2 := Cpu{
		Id:           2,
		GeneralQueue: cpuQueue,
		DirectQueue:  make(chan Process),
		Wg:           &cpuWg,

		IsBusy:          false,
		CurProcess:      Process{},
		CurProcessMutex: sync.Mutex{},
	}

	scheduler := Scheduler{
		Gen1: &gen1,
		Gen2: &gen2,
		Cpu1: &cpu1,
		Cpu2: &cpu2,

		SchedulerQueue: schedulerQueue,
		CpuQueue:       cpuQueue,
		GenWg:          &genWg,
	}

	flag.IntVar(&gen1.ProcessesToGenerate, "g1p", 15, "number of processes for GEN1 to generate")
	flag.IntVar(&gen1.MinProcessGenerationTime, "g1m", 50, "min GEN1 process generation time")
	flag.IntVar(&gen1.MaxProcessGenerationTime, "g1M", 300, "max GEN1 process generation time")

	flag.IntVar(&gen2.ProcessesToGenerate, "g2p", 15, "number of processes for GEN1 to generate")
	flag.IntVar(&gen2.MinProcessGenerationTime, "g2m", 50, "min GEN1 process generation time")
	flag.IntVar(&gen2.MaxProcessGenerationTime, "g2M", 300, "max GEN1 process generation time")

	flag.IntVar(&cpu1.MinProcessingTime, "c1m", 60, "min CPU1 processing time")
	flag.IntVar(&cpu1.MaxProcessingTime, "c1M", 200, "max CPU1 processing time")

	flag.IntVar(&cpu2.MinProcessingTime, "c2m", 30, "min CPU1 processing time")
	flag.IntVar(&cpu2.MaxProcessingTime, "c2M", 100, "max CPU1 processing time")

	printHelp := flag.Bool("help", false, "print this message")

	flag.Parse()

	if *printHelp {
		flag.Usage()
		os.Exit(0)
	}

	go cpu1.Run()
	go cpu2.Run()
	go gen1.Run()
	go gen2.Run()

	scheduler.Run()
	cpuWg.Wait()
}
