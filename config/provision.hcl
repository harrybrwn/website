s3 {
	access_key = "root"
	secret_key = "minio-testbed001"
	// endpoint   = "localhost:9000"

	bucket "loki-logs" {}
	bucket "docker-registry" {}

	user "loki" {
		access_key = "loki"
		secret_key = "minio-testbed001-loki"
		policies   = ["loki"]
	}
}

db {
	host     = "localhost"
	port     = "5432"
	root_user = "harrybrwn"
	password = "testbed01"

	database "grafana" {
		owner = "grafana"
	}
}