package config

const (

	//Local influxdb data
	BUCKET      = ""
	INFLUX_URL  = "http://localhost:8086"
	LOCAL_ORG   = ""
	LOCAL_TOKEN = ""

	//Linux influxdb data (same bucket name and url)
	REMOTE_ORG   = ""
	REMOTE_TOKEN = ""

	//Local postgres data
	POSTGRES_USERNAME       = "postgres"
	LOCAL_POSTGRES_PASSWORD = ""
	POSTGRES_HOST           = "localhost"
	POSTGRES_PORT           = "5432"
	POSTGRES_DB_NAME        = ""

	//Remote postgres data (same username, host, port and db name)
	REMOTE_POSTGRES_PASSWORD = "password"
)
