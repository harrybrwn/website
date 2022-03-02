import pytest

import time
from datetime import datetime
import requests

import config
from models import Token

NANOSECOND  = 1
MICROSECOND = 1000 * NANOSECOND
MILLISECOND = 1000 * MICROSECOND
SECOND      = 1000 * MILLISECOND


def test_delete_nonexistent_invite(admin_token: Token):
	res = requests.delete(f"http://{config.host}/api/invite/123")
	assert not res.ok
	res = requests.delete(
		f"http://{config.host}/api/invite/123",
		headers={"Authorization": admin_token.header()},
	)
	assert not res.ok


def test_create_invite(admin_token: Token):
	res = requests.post(
		f"http://{config.host}/api/invite/create",
		headers={"Authorization": admin_token.header()},
	)
	assert res.ok
	j = res.json()
	assert j["ttl"] > 0


def test_create_invite_admin(admin_token: Token):
	res = requests.post(
		f"http://{config.host}/api/invite/create",
		headers={"Authorization": admin_token.header()},
		json={
			"roles": [
				"default",
				"family"
			],
			"ttl": 15,
			"expires": SECOND * 5
		},
	)
	assert res.ok
	j = res.json()
	assert j["ttl"] == 15
	assert j["roles"] == ["default", "family"]
	res = requests.get(f"http://{config.host}{j['path']}")
	assert res.ok


def test_invite_timeout(admin_token: Token):
	res = requests.post(
		f"http://{config.host}/api/invite/create",
		headers={"Authorization": admin_token.header()},
		json={
			"expires": SECOND * 5,
		},
	)
	assert res.ok
	j = res.json()
	print(j["expires_at"])
	print(type(j["expires_at"]))
	expires_at = datetime.strptime(j["expires_at"], '%Y-%m-%dT%H:%M:%S.%fZ')
	assert expires_at > datetime.now()
	print(expires_at)

	res = requests.get(f"http://{config.host}{j['path']}", headers={"accept":"text/html"})
	assert res.ok
	assert res.headers.get("Content-Type") == "text/html"
	time.sleep(6)
	res = requests.get(f"http://{config.host}{j['path']}", headers={"accept":"text/html"})
	print(res)
	print(res.text)
	assert not res.ok
