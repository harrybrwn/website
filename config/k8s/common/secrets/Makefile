help:
	@echo help?

# Staging is more likely to delete the master key
stg:
	./from-env-file.sh -env stg stg/cf-dns-token.env
	./from-env-file.sh -env stg stg/sendgrid.env
	./from-env-file.sh -env stg stg/sendgrid-mastodon.env

.PHONY: stg
