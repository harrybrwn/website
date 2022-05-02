import pytest

import time
from datetime import datetime
import requests

import config
from models import Role, Invite, Token

NANOSECOND  = 1
MICROSECOND = 1000 * NANOSECOND
MILLISECOND = 1000 * MICROSECOND
SECOND      = 1000 * MILLISECOND


def test_delete_nonexistent_invite(admin_token: Token):
	res = requests.delete(f"{config.scheme}://{config.host}/api/invite/123")
	assert not res.ok
	res = requests.delete(
		f"{config.scheme}://{config.host}/api/invite/123",
		headers={"Authorization": admin_token.header()},
	)
	assert not res.ok


def test_create_invite(admin_token: Token):
	res = requests.post(
		f"{config.scheme}://{config.host}/api/invite/create",
		headers={"Authorization": admin_token.header()},
	)
	assert res.ok
	j = res.json()
	assert j["ttl"] > 0


def test_create_invite_admin(admin_token: Token):
	res = requests.post(
		f"{config.scheme}://{config.host}/api/invite/create",
		headers={"Authorization": admin_token.header()},
		json={
			"roles": [
				"default",
				"family"
			],
			"ttl": 15,
			"expires": SECOND * 5,
			"receiver_name": "jerry smith",
		},
	)
	assert res.ok
	j = res.json()
	assert j["ttl"] == 15
	assert j["roles"] == [Role.DEFAULT.value, Role.FAMILY.value]
	assert j["receiver_name"] == "jerry smith"
	res = requests.get(f"{config.scheme}://{config.host}{j['path']}")
	assert res.ok
	assert res.headers.get("content-type") == "text/html"


def test_invite_timeout(admin_token: Token):
	res = requests.post(
		f"{config.scheme}://{config.host}/api/invite/create",
		headers={"Authorization": admin_token.header()},
		json={
			"timeout": SECOND * 1,
		},
	)
	assert res.ok
	inv = Invite.from_json(res.json())
	assert inv.expires_at > datetime.now()

	res = requests.get(f"{config.scheme}://{config.host}{inv.path}", headers={"accept":"text/html"})
	assert res.ok
	assert res.headers.get("Content-Type") == "text/html"

	time.sleep(1)
	res = requests.get(f"{config.scheme}://{config.host}{inv.path}", headers={"accept":"text/html"})
	assert not res.ok
