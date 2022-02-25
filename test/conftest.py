import pytest

from models import User


@pytest.fixture(scope="module")
def user():
    u = User(
        "testuser",
        "test@harrybrwn.com",
        "password1",
        ["default"],
    )
    u.create()
    yield u
    u.delete()


@pytest.fixture(scope="module")
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


@pytest.fixture(scope="module")
def user_token(user: User):
    user.login()
    return user.token


@pytest.fixture(scope="module")
def admin_token(admin: User):
    admin.login()
    return admin.token
