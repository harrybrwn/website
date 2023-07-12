help:
	@echo 'Usage'
	@echo '  make [target]'
	@echo
	@echo 'Targets'
	@echo '  ping     ping the hosts'
	@echo '  stg      bootstrap the staging environment'
	@echo '  prd      bootstrap the production environment'
	@echo '  registry configure and start the remote container registry'

ping:
	ansible -i inventory.yml -m ping k3s_cluster

stg:
	ansible-playbook -i stg-inventory.yml --vault-password-file vault-password.txt stg.yml

prd:
	ansible-playbook -i inventory.yml --vault-password-file vault-password.txt site.yml

registry:
	ansible-playbook -i inventory.yml --vault-password-file vault-password.txt registry.yml

destroy-stg:
	ansible-playbook -i stg-inventory.yml --vault-password-file vault-password.txt playbooks/destroy-staging.yml

.PHONY: ping stg prd registry
