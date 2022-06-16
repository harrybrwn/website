package main

import "context"

type DockerConfig struct {
	CACert       string
	ServerCert   string
	ServerKey    string
	DaemonConfig map[string]any
}

func (dc *DockerConfig) Provision(ctx context.Context) error {
	// TODO
	// - copy over daemon.json
	// - copy over (and maybe generate) certificates/keys
	// - copy container registry certificate to /etc/docker/certs.d/
	// - setup docker context to access remote machines
	// - replace default ingress if it has the same subnet as LAN
	return nil
}
