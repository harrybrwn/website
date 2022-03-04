from __future__ import annotations
import datetime
from typing import List

import requests
import bcrypt

from db import db
import config


class Token:
    token: str
    type: str
    expires: int
    refresh_token: str

    def __init__(self, blob):
        self.token = blob["token"]
        self.type = blob["token_type"]
        self.expires = blob["expires"]
        self.refresh_token = blob["refresh_token"]

    def header(self) -> str:
        return f"{self.type} {self.token}"


class User:
    id: int
    uuid: str
    username: str
    email: str
    password: str
    pw_hash: bytes
    roles: List[str]
    created_at: datetime.datetime
    updated_at: datetime.datetime
    token: Token

    def __init__(self, username: str, email: str, password: str, roles: List[str]):
        self.username = username
        self.email = email
        self.password = password
        self.pw_hash = bcrypt.hashpw(password.encode('utf8'), bcrypt.gensalt(10))
        self.roles = roles
        self.id = 0
        self.uuid = ''
        self.created_at = None
        self.updated_at = None
        self.token = None

    def create(self, conn=None):
        if conn is None:
            conn = db
        cur = conn.cursor()
        cur.execute('''INSERT INTO "user"
            (uuid, username, email, pw_hash, roles, totp_secret)
            VALUES (
                uuid_in(md5(random()::text || clock_timestamp()::text)::cstring),
                %s, %s, %s, %s, ''
            ) RETURNING id, uuid, created_at, updated_at''',
            (self.username, self.email, self.pw_hash, self.roles),
        )
        res = cur.fetchone()
        self.id = res[0]
        self.uuid = res[1]
        self.created_at = res[2]
        self.updated_at = res[3]
        conn.commit()
        cur.close()

    def delete(self):
        cur = db.cursor()
        cur.execute('''DELETE FROM "user"
            WHERE
                id = %s AND uuid = %s AND username = %s AND email = %s''',
            (self.id, self.uuid, self.username, self.email))
        db.commit()
        cur.close()


    def login(self):
        res = requests.post(f"http://{config.host}/api/token", json={
            "username": self.username,
            "email": self.email,
            "password": self.password,
        })
        if not res.ok:
            raise requests.RequestException(f"status {res.status_code}: could not login")
        self.token = Token(res.json())


    def __str__(self):
        return f'User({self.id}, "{self.uuid}", "{self.username}", "{self.email}", {self.roles})'


class Invite:
    path: str
    created_by: str
    expires_at: datetime.datetime
    email: str
    roles: List[str]
    ttl: int

    @staticmethod
    def from_json(j) -> Invite:
        return Invite(
            j.get("path"),
            j.get("created_by"),
            datetime.datetime.strptime(j["expires_at"], '%Y-%m-%dT%H:%M:%S.%fZ'),
            j.get("email"),
            j.get("roles"),
            j.get("ttl"),
        )

    def __init__(self, path: str, created_by: str, expires_at: datetime.datetime, email: str, roles: List[str], ttl: int):
        self.path = path
        self.created_by = created_by
        self.expires_at = expires_at
        self.email = email
        self.roles = roles
        self.ttl = ttl

