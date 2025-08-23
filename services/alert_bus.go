package services

import (
	"fmt"
	"time"

	"backend/models"
	"gorm.io/gorm"
)

type alertDeps struct {
	db  *gorm.DB
	rt  *RealtimeHub
	ps  *PushService
}

var _alert alertDeps

func InitAlertDeps(db *gorm.DB, rt *RealtimeHub, ps *PushService) {
	_alert = alertDeps{db: db, rt: rt, ps: ps}
}

func EmitAlert(userID uint, typ, message string) { // safe to call anywhere
	if _alert.db == nil { return } // not initialized
	a := &models.Alert{UserID: userID, Type: typ, Message: message, CreatedAt: time.Now()}
	_ = _alert.db.Create(a).Error

	if _alert.rt != nil {
		_alert.rt.BroadcastAlert(userID, map[string]any{
			"kind":  "alert.created",
			"alert": a,
		})
	}
	if _alert.ps != nil {
		_alert.ps.PushToUser(userID, "New Alert", message, map[string]string{
			"type": typ, "alertId": fmt.Sprintf("%d", a.ID),
		})
	}
}
