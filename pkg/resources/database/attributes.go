package database

const (
	Name = "exoscale_database"
)

var (
	ServicesList = []string{
		"kafka",
		"mysql",
		"pg",
		"redis",
		"opensearch",
		"grafana",
	}
)
