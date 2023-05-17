help:
	echo 'Usage'
	echo '  make [target]'
	echo
	echo 'Targets'
	echo '  stg   bootstrap the staging environment'
	echo '  prd   bootstrap the production environment'

stg:
	ansible-playbook -i stg-inventory.yml --vault-password-file vault-password.txt stg.yml

prd:
	ansible-playbook -i inventory.yml --vault-password-file vault-password.txt site.yml

destroy-stg:
	ansible-playbook -i stg-inventory.yml --vault-password-file vault-password.txt playbooks/destroy-staging.yml
