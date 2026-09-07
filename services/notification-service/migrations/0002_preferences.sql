-- Migration 0002: notification_preferences
-- Per-(user, type, channel) preferences with quiet hours and digest mode.
CREATE TABLE IF NOT EXISTS notification.notification_preferences (
    user_id TEXT NOT NULL,
    notif_type TEXT NOT NULL,
    channel TEXT NOT NULL
        CHECK (channel IN ('in_app','email','sms','push','slack','discord','telegram','webhook')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    quiet_start INT CHECK (quiet_start >= 0 AND quiet_start <= 23),
    quiet_end INT CHECK (quiet_end >= 0 AND quiet_end <= 23),
    digest_mode TEXT NOT NULL DEFAULT 'none' CHECK (digest_mode IN ('none','daily','weekly')),
    PRIMARY KEY (user_id, notif_type, channel)
);

CREATE INDEX IF NOT EXISTS idx_prefs_user ON notification.notification_preferences(user_id);