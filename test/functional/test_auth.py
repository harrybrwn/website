import pytest

import requests


def test_homepage():
    res = requests.get("http://web:8080/")
    assert res.ok


def test_login(user):
    req = {
        "email": user.email,
        "password": user.password,
    }
    res = requests.post("http://web:8080/api/token", json=req)
    assert res.ok
    j = res.json()
    assert "token" in j
    assert "refresh_token" in j
    assert "token_type" in j


def test_runtime(admin_token):
    t = admin_token
    res = requests.get("http://web:8080/api/runtime", headers={
        "Authorization": f'{t["token_type"]} {t["token"]}',
    })
    assert res.ok
