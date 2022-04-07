import pytest
import base64
import json

import requests
from models import Role, Token
from config import host
import time


def b64decode(s: str):
    b = s.encode('utf-8')
    return base64.decodebytes(b + b'=' * (-len(b) % 4))


def test_homepage():
    assert False
    res = requests.get(f"http://{host}/")
    assert res.ok
    assert res.status_code == 200


def test_admin_page_logedin(admin_token: Token):
    res = requests.get(f"http://{host}/admin", headers={"Authorization": admin_token.header()})
    assert res.ok
    assert res.status_code == 200


def test_robots_txt():
    res = requests.get(f"http://{host}/robots.txt")
    assert res.ok
    assert res.status_code == 200


def test_admin_page_failure():
    res = requests.get(f"http://{host}/admin")
    assert not res.ok
    assert res.status_code == 401


def test_runtime_as_admin(admin_token: Token):
    res = requests.get(f"http://{host}/api/runtime", headers={"Authorization": admin_token.header()})
    assert res.ok
    info = res.json()
    assert info["name"] == "Harry Brown"
    assert "age" in info
    assert "uptime" in info
    assert "birthday" in info
    assert "build" in info
    assert "dependencies" in info
    assert "module" in info


def test_runtime_as_user(user_token: Token):
    res = requests.get(f"http://{host}/api/runtime", headers={"Authorization": user_token.header()})
    assert not res.ok
    assert res.status_code >= 400 and res.status_code < 500


def test_token_revoke(admin_token: Token):
    res = requests.post(
        f"http://{host}/api/revoke",
        headers={"Authorization": admin_token.header()},
        json={"refresh_token": admin_token.refresh_token}
    )
    assert res.ok
    assert res.status_code == 200

    res = requests.post(f"http://{host}/api/refresh", json={
        "refresh_token": admin_token.refresh_token,
    })
    assert not res.ok
    assert res.status_code == 401


# Generating JWT tokens relies on the current time. So keep this last so that
# the session scoped token fixture is generated longer than a second before this
# test runs.
def test_token_refresh(user_token: Token):
    # generating two JWT tokens in the same second will result in the same "iat"
    # field and they will be the same.
    time.sleep(1.0)
    res = requests.post(f"http://{host}/api/refresh", json={
        "refresh_token": user_token.refresh_token,
    })
    assert res.ok
    tok = Token(res.json())
    assert tok.token != user_token.token
    assert tok.expires > user_token.expires
    assert tok.refresh_token == user_token.refresh_token


def test_token_claims_admin(admin_token: Token):
    parts = admin_token.token.split(".")
    assert len(parts) == 3
    meta = json.loads(b64decode(parts[0]))
    assert meta is not None
    claims = json.loads(b64decode(parts[1]))
    assert Role.ADMIN.value in claims["roles"]
    assert claims['iss'] == 'harrybrwn.com'
    assert claims['id'] != 0
    assert len(claims['uuid']) > 0


def test_token_claims_user(user_token: Token):
    parts = user_token.token.split(".")
    assert len(parts) == 3
    meta = json.loads(b64decode(parts[0]))
    assert meta is not None
    claims = json.loads(b64decode(parts[1]))
    assert Role.DEFAULT.value in claims['roles']
    assert claims['iss'] == 'harrybrwn.com'
    assert claims['id'] != 0
    assert len(claims['uuid']) > 0
