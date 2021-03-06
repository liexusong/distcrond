package scheduler

import (
//	"time"
	"github.com/martin-helmich/distcrond/container"
	. "github.com/martin-helmich/distcrond/domain"
	"github.com/martin-helmich/distcrond/runner"
	"github.com/martin-helmich/distcrond/logging"
//	"sync/atomic"
	"github.com/robfig/cron"
)

type Scheduler struct {
	jobContainer *container.JobContainer
	nodeContainer *container.NodeContainer
	runner runner.JobRunner
	abort chan bool

	Done chan bool
}

type JobWrapper struct {
	runner runner.JobRunner
	job *Job
	semaphore chan bool
}

func NewScheduler(jobs *container.JobContainer, nodes *container.NodeContainer, runner runner.JobRunner) *Scheduler {
	return &Scheduler {
		jobs,
		nodes,
		runner,
		make(chan bool),
		make(chan bool),
	}
}

func (s *Scheduler) Abort() {
	s.abort <- true
}

//func (s *Scheduler) nextRunDate(job *Job, now time.Time) time.Time {
//	reference := job.Schedule.Reference
//	todayReference := time.Date(now.Year(), now.Month(), now.Day(), reference.Hour(), reference.Minute(), reference.Second(), reference.Nanosecond(), reference.Location())
//
//	if todayReference.Before(now) {
//		for todayReference.Before(now) {
//			todayReference = todayReference.Add(job.Schedule.Interval)
//		}
//	} else {
//		for todayReference.After(now) {
//			todayReference = todayReference.Add(-job.Schedule.Interval)
//		}
//		todayReference = todayReference.Add(job.Schedule.Interval)
//	}
//
//	return todayReference
//}

func (s *Scheduler) Run() {
	logging.Info("Starting scheduler")

	var jobCount   int               = s.jobContainer.Count()
	var semaphores []chan bool       = make([]chan bool, jobCount)
//	var tickers    chan *time.Ticker = make(chan *time.Ticker, jobCount)
//	var now        time.Time         = time.Now()

//	var startedTickers int64 = 0

//	withLock := func(f func(), i int) {
//		semaphores[i] <- true
//		f()
//		<-semaphores[i]
//	}

//	var start []time.Time = make([]time.Time, jobCount)
//	for i := 0; i < jobCount; i ++ {
//		start[i] = s.nextRunDate(s.jobContainer.Get(i), now)
//	}

	cron := cron.New()

	for i := 0; i < jobCount; i ++ {
		job := s.jobContainer.Get(i)
		semaphores[i] = make(chan bool, 1)

		cmd := JobWrapper{runner: s.runner, job: job, semaphore: semaphores[i]}
		cron.Schedule(job.Schedule, cmd)

//		go func(job *Job, i int) {
//			wait := start[i].Sub(now)
//			logging.Debug("Next execution of %s scheduled for %s, waiting %s", job.Name, start[i].String(), wait.String())
//			<- time.After(wait)
//
//			atomic.AddInt64(&startedTickers, 1)
//
//			ticker := time.NewTicker(job.Schedule.Interval)
//			tickers <- ticker
//
//			logging.Debug("Started timer for %s", job.Name)
//
//			runJob := func(t time.Time) {
//				withLock(func() {
//					logging.Notice("Executing job %s at %s", job.Name, t)
//					s.runner.Run(job)
//				}, i)
//			}
//
//			runJob(time.Now())
//
//			for t := range ticker.C {
//				runJob(t)
//			}
//		}(job, i)
	}

	cron.Start()

	select {
	case <- s.abort:
		logging.Notice("Aborting")

//		logging.Debug("Stopping tickers...")
//		for i := atomic.LoadInt64(&startedTickers); i > 0; i -- {
//			(<- tickers).Stop()
//		}

		logging.Debug("Stopping scheduler...")
		cron.Stop()

		logging.Notice("Waiting for running jobs...")
		for i := 0; i < jobCount; i ++ {
			semaphores[i] <- true
		}

		logging.Debug("Done")
		s.Done <- true
	}
}

func (w JobWrapper) Run() {
	w.semaphore <- true
	w.runner.Run(w.job)
	<- w.semaphore
}