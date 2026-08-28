package controllers

import (
	"time"

	"github.com/astaxie/beego"
)

func runJobOnce() {
	summary, err := refreshOrdersAndStatistics()
	if err != nil {
		beego.Error("Daily job failed:", err)
		return
	}

	beego.Info(
		"Daily statistics job finished. successful_orders:",
		summary.SuccessfulOrders,
		" failed_orders:",
		summary.FailedOrders,
	)
	if err := generateDueDateNotifications(); err != nil {
		beego.Error("Due-date notification job failed:", err)
	}
}

func StartDailyJob() {
	go func() {
		for {
			now := time.Now()

			next := time.Date(
				now.Year(), now.Month(), now.Day(),
				2, 0, 0, 0, now.Location(),
			)

			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			time.Sleep(next.Sub(now))

			beego.Info("Running daily job...")
			runJobOnce()
		}
	}()
}
