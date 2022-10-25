job "ping" {
  datacenters = ["dc1"]

  type = "service"

  group "example" {
    task "ping" {
      driver = "exec"

      config {
        command = "/bin/ping"
        args    = ["-c", "1000", "-i", "5", "google.com"]
      }
    }
  }
}