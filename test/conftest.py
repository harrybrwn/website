import pytest

import requests
from models import User


@pytest.fixture
def user():
    username = "testuser"
    u = User(
        username,
        "test@harrybrwn.com",
        "password1",
        ["default"],
    )
    u.create()
    yield u
    u.delete()


@pytest.fixture
def admin():
    u = User(
        "admin_user",
        "admin@example.com",
        "onetwothreefourfive",
        ["admin"],
    )
    u.create()
    yield u
    u.delete()


@pytest.fixture
def token(user):
    res = requests.post("http://web:8080/api/token", json={
        "username": user.username,
        "email": user.email,
        "password": user.password,
    })
    return res.json()


@pytest.fixture
def admin_token(admin):
    res = requests.post("http://web:8080/api/token", json={
        "username": admin.username,
        "email": admin.email,
        "password": admin.password,
    })
    return res.json()
