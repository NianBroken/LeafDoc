// 本文件串行消费周期、手动触发和设置变化事件，调用管理器创建快照。
// 每轮事件结束后重新计算下一时间，不按任务开始前的固定时间线补跑错过的周期。
package snapshot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	// 将时区数据库编入程序，让每日调度不只依赖宿主机提供的时区文件。
	_ "time/tzdata"
)

// Scheduler 使用单个工作协程执行任务。两个容量为一的事件通道合并待处理通知，
// done 表示停止请求，等待组用于确认该协程已退出。
type Scheduler struct {
	manager   *Manager
	manual    chan struct{}
	updated   chan struct{}
	done      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// NewScheduler 读取启动标记，按需预先放入一个手动事件，再启动工作协程。
// 启动标记单独判断，即使 Enabled 为假也会按 RunOnStartup 执行启动任务。
func NewScheduler(manager *Manager) *Scheduler {
	s := &Scheduler{
		manager: manager,
		manual:  make(chan struct{}, 1),
		updated: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
	initialSettings := manager.Settings()
	if initialSettings.RunOnStartup {
		s.manual <- struct{}{}
	}
	s.wg.Add(1)
	go s.run()
	return s
}

// run 每轮创建计时器并等待一个事件。任务同步执行，期间不消费设置变化或停止事件。
// 管理器收到后台上下文，所以 Stop 不会取消正在执行的归档，只等待它返回。
func (s *Scheduler) run() {
	defer s.wg.Done()
	for {
		timer, timerC := s.nextTimer()
		select {
		case <-timerC:
			if err := s.manager.RunOnce(context.Background()); err != nil {
				slog.Error("snapshot failed", "error", err)
			}
		case <-s.manual:
			if err := s.manager.RunOnce(context.Background()); err != nil {
				slog.Error("snapshot failed", "error", err)
			}
		case <-s.updated:
			// 设置事件本身不执行快照，结束本轮后用最新设置重建计时器。
		case <-s.done:
			stopTimer(timer)
			return
		}
		stopTimer(timer)
	}
}

// nextTimer 在周期未启用或时间计算失败时返回空通道，使 select 的计时分支禁用。
// 已过的计算时间使用零延迟，不在此循环追赶此前错过的调度点。
func (s *Scheduler) nextTimer() (*time.Timer, <-chan time.Time) {
	settings := s.manager.Settings()
	if !settings.Enabled {
		return nil, nil
	}
	next, err := nextScheduledRun(time.Now(), settings)
	if err != nil {
		return nil, nil
	}
	delay := time.Until(next)
	if delay < 0 {
		delay = 0
	}
	timer := time.NewTimer(delay)
	return timer, timer.C
}

// nextScheduledRun 计算严格晚于 now 的每日时区时刻，等于调度时间时选择下一天。
// 其他模式走间隔解析，至少一分钟；此函数不检查 Enabled，也不代替完整设置校验。
func nextScheduledRun(now time.Time, settings Settings) (time.Time, error) {
	var next time.Time
	switch settings.Mode {
	case scheduleModeDaily:
		location, err := time.LoadLocation(settings.TimeZone)
		if err != nil {
			return time.Time{}, err
		}
		// 按目标时区的日历构造日期，而不是简单给 UTC 时间加二十四小时。
		localNow := now.In(location)
		next = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), settings.Hour, settings.Minute, 0, 0, location)
		if !next.After(localNow) {
			next = time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, settings.Hour, settings.Minute, 0, 0, location)
		}
	default:
		interval, err := time.ParseDuration(settings.Interval)
		if err != nil || interval < minSnapshotInterval {
			return time.Time{}, fmt.Errorf("invalid interval %q", settings.Interval)
		}
		return now.Add(interval), nil
	}
	return next, nil
}

// stopTimer 接受空计时器，停止失败时尝试非阻塞取走已送达的计时通知。
func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

// UpdateSettings 非阻塞登记一次重算通知，已有待处理通知时合并，不排入多项。
func (s *Scheduler) UpdateSettings() {
	select {
	case s.updated <- struct{}{}:
	default:
	}
}

// TriggerNow 尝试登记一个手动任务，返回值表示是否成功入队，不代表归档已完成。
// 当前任务执行时仍可排队一个后续任务；此入口不检查调度器是否已经停止。
func (s *Scheduler) TriggerNow() bool {
	select {
	case s.manual <- struct{}{}:
		return true
	default:
		return false
	}
}

// Stop 只关闭一次停止通道，然后等待工作协程结束。
// 当前执行不被取消；多个事件同时就绪时，select 不保证停止事件先被选中。
func (s *Scheduler) Stop() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	s.wg.Wait()
}
