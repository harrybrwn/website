import pytest

import os
import redis
from models import User, Role


@pytest.fixture(scope="session")
def rdb() -> redis.Redis:
    return redis.Redis(
        host="redis",
        port=6379,
        db=0,
        password=os.getenv("REDIS_PASSWORD"),
    )


@pytest.fixture(scope="session", autouse=True)
def clear_cache(rdb: redis.Redis):
    rdb.flushall(False)


@pytest.fixture(scope="module")
def user():
    u = User(
        "testuser",
        "test@harrybrwn.com",
        "password1",
        [Role.DEFAULT],
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
        [Role.ADMIN],
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
