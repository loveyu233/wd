package wd

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

var InsCronJob *CronConfig

type CronConfig struct {
	location              *time.Location                                   // 时区
	beforeJobRuns         func(jobID uuid.UUID, jobName string)            // 运行前
	afterJobRuns          func(jobID uuid.UUID, jobName string)            // 运行后
	afterJobRunsWithError func(jobID uuid.UUID, jobName string, err error) // 出错
	options               []gocron.SchedulerOption
	Scheduler             gocron.Scheduler
}

// RunJob 用来在调度器中注册一个任务。
func (cron *CronConfig) RunJob(df gocron.JobDefinition, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	return cron.Scheduler.NewJob(df, task, options...)
}

// redisKey 用来构建任务锁的 Redis 键名。
func (cron *CronConfig) redisKey(id any) string {
	return fmt.Sprintf("cron-%v-lock", id)
}

// RunJobTheOne 用来在持有 Redis 锁时才注册任务，避免重复调度。
func (cron *CronConfig) RunJobTheOne(id any, df gocron.JobDefinition, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	if InsRedis == nil {
		return nil, redisClientNilErr()
	}
	if err := InsRedis.NewLock(cron.redisKey(id)).TryLock(); err == nil {
		return cron.Scheduler.NewJob(df, task, options...)
	}
	return nil, nil
}

// RunJobEveryDuration 用来按固定间隔周期执行任务。
func (cron *CronConfig) RunJobEveryDuration(duration time.Duration, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	return cron.Scheduler.NewJob(
		gocron.DurationJob(duration),
		task,
		options...,
	)
}

// RunJobEveryDurationTheOne 用来在加锁后以固定间隔执行任务。
func (cron *CronConfig) RunJobEveryDurationTheOne(id any, duration time.Duration, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	if InsRedis == nil {
		return nil, redisClientNilErr()
	}
	if err := InsRedis.NewLock(cron.redisKey(id)).TryLock(); err == nil {
		return cron.Scheduler.NewJob(
			gocron.DurationJob(duration),
			task,
			options...,
		)
	}
	return nil, nil
}

// RunJobAtTime 用来在指定时间运行一次任务。
func (cron *CronConfig) RunJobAtTime(time time.Time, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	return cron.Scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(time)), task, options...)
}

// RunJobAtTimeTheOne 用来加锁后在指定时间运行一次任务。
func (cron *CronConfig) RunJobAtTimeTheOne(id any, time time.Time, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	if InsRedis == nil {
		return nil, redisClientNilErr()
	}
	if err := InsRedis.NewLock(cron.redisKey(id)).TryLock(); err == nil {
		return cron.Scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(time)), task, options...)
	}
	return nil, nil

}

// RunJobAtTimes 用来在多个指定时间各运行一次任务。
func (cron *CronConfig) RunJobAtTimes(times []time.Time, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	return cron.Scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTimes(times...)), task, options...)
}

// RunJobAtTimesTheOne 用来在持锁情况下在多个时间执行任务。
func (cron *CronConfig) RunJobAtTimesTheOne(id any, times []time.Time, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	if InsRedis == nil {
		return nil, redisClientNilErr()
	}
	if err := InsRedis.NewLock(cron.redisKey(id)).TryLock(); err == nil {
		return cron.Scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTimes(times...)), task, options...)
	}
	return nil, nil
}

// RunJobEveryDay 用来按每天固定时间段执行任务。
func (cron *CronConfig) RunJobEveryDay(hours, minutes, seconds, interval uint, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	return cron.Scheduler.NewJob(
		gocron.DailyJob(interval, gocron.NewAtTimes(
			gocron.NewAtTime(hours, minutes, seconds),
		)),
		task,
		options...,
	)
}

// RunJobEveryDayTheOne 用来在加锁后每天固定时刻执行任务。
func (cron *CronConfig) RunJobEveryDayTheOne(id any, hours, minutes, seconds, interval uint, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	if InsRedis == nil {
		return nil, redisClientNilErr()
	}
	if err := InsRedis.NewLock(cron.redisKey(id)).TryLock(); err == nil {
		return cron.Scheduler.NewJob(
			gocron.DailyJob(interval, gocron.NewAtTimes(
				gocron.NewAtTime(hours, minutes, seconds),
			)),
			task,
			options...,
		)
	}
	return nil, nil
}

// RunJobCrontab 用来根据 cron 表达式调度任务。
func (cron *CronConfig) RunJobCrontab(crontab string, withSeconds bool, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	return cron.Scheduler.NewJob(
		gocron.CronJob(crontab, withSeconds),
		task,
		options...,
	)
}

// RunJobCrontabTheOne 用来在锁定后按 cron 表达式调度任务。
func (cron *CronConfig) RunJobCrontabTheOne(id any, crontab string, withSeconds bool, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	if InsRedis == nil {
		return nil, redisClientNilErr()
	}
	if err := InsRedis.NewLock(cron.redisKey(id)).TryLock(); err == nil {
		return cron.Scheduler.NewJob(
			gocron.CronJob(crontab, withSeconds),
			task,
			options...,
		)
	}
	return nil, nil
}

type CronOption func(*CronConfig)

// WithLocation 用来指定调度器使用的时区。
func WithLocation(loc *time.Location) CronOption {
	return func(c *CronConfig) {
		c.location = loc
	}
}

// WithBeforeJobRuns 用来设置任务执行前的回调。
func WithBeforeJobRuns(beforeJobRuns func(jobID uuid.UUID, jobName string)) CronOption {
	return func(c *CronConfig) {
		c.beforeJobRuns = beforeJobRuns
	}
}

// WithAfterJobRuns 用来设置任务执行后的回调。
func WithAfterJobRuns(afterJobRuns func(jobID uuid.UUID, jobName string)) CronOption {
	return func(c *CronConfig) {
		c.afterJobRuns = afterJobRuns
	}
}

// WithAfterJobRunsWithError 用来设置任务出错时的回调。
func WithAfterJobRunsWithError(afterJobRunsWithError func(jobID uuid.UUID, jobName string, err error)) CronOption {
	return func(c *CronConfig) {
		c.afterJobRunsWithError = afterJobRunsWithError
	}
}

// WithCronJobs 用来追加自定义的调度器选项。
func WithCronJobs(options ...gocron.SchedulerOption) CronOption {
	return func(c *CronConfig) {
		c.options = append(c.options, options...)
	}
}

// InitCronJob 用来初始化 gocron 调度器并保存全局实例。
func InitCronJob(options ...CronOption) error {
	var cron = &CronConfig{
		options: make([]gocron.SchedulerOption, 0),
	}
	for _, opt := range options {
		opt(cron)
	}

	if cron.location == nil {
		cron.location = ShangHaiTimeLocation
	}
	cron.options = append(cron.options, gocron.WithLocation(cron.location))

	var eventListeners []gocron.EventListener
	if cron.afterJobRuns != nil {
		eventListeners = append(eventListeners, gocron.AfterJobRuns(cron.afterJobRuns))
	}
	if cron.beforeJobRuns != nil {
		eventListeners = append(eventListeners, gocron.BeforeJobRuns(cron.beforeJobRuns))
	}
	if cron.afterJobRunsWithError != nil {
		eventListeners = append(eventListeners, gocron.AfterJobRunsWithError(cron.afterJobRunsWithError))
	}
	if len(eventListeners) > 0 {
		cron.options = append(cron.options, gocron.WithGlobalJobOptions(
			gocron.WithEventListeners(eventListeners...),
		))
	}

	scheduler, err := gocron.NewScheduler(cron.options...)
	if err != nil {
		return err
	}

	cron.Scheduler = scheduler
	InsCronJob = cron
	return nil
}

// Start 用来启动调度器开始运行任务。
func (cron *CronConfig) Start() {
	cron.Scheduler.Start()
}

// Stop 用来优雅关闭调度器。
func (cron *CronConfig) Stop() error {
	if err := cron.Scheduler.Shutdown(); err != nil {
		return err
	}
	return nil
}
