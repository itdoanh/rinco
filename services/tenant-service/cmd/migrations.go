package main

import (
	mig "github.com/itdoanh/rinco/services/tenant-service/cmd/migrations"
)

// Migration is the ordered list of SQL migrations run on startup.
var Migrations = []struct {
	Name string
	SQL  string
}{
	{"0001_init", mig.SQL_0001_init},
	{"0002_domains_settings", mig.SQL_0002_domains_settings},
	{"0003_usage", mig.SQL_0003_usage},
}
