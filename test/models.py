from typing import List
from db import db

import bcrypt


class User:
    id: int
    uuid: str
    username: str
    email: str
    password: str
    pw_hash: bytes
    roles: List[str]

    def __init__(self, username: str, email: str, password: str, roles: List[str]):
        self.username = username
        self.email = email
        self.password = password
        self.pw_hash = bcrypt.hashpw(password.encode('utf8'), bcrypt.gensalt(10))
        self.roles = roles
        self.id = 0
        self.uuid = ''

    def create(self):
        cur = db.cursor()
        cur.execute('''INSERT INTO "user"
            (uuid, username, email, pw_hash, roles, totp_secret)
            VALUES (
                uuid_in(md5(random()::text || clock_timestamp()::text)::cstring),
                %s, %s, %s, %s, ''
            ) RETURNING id, uuid''',
            (self.username, self.email, self.pw_hash, self.roles),
        )
        res = cur.fetchone()
        self.id = res[0]
        self.uuid = res[1]
        db.commit()
        cur.close()

    def delete(self):
        cur = db.cursor()
        cur.execute('''DELETE FROM "user"
            WHERE
                id = %s AND uuid = %s AND username = %s AND email = %s''',
            (self.id, self.uuid, self.username, self.email))
        db.commit()
        cur.close()


    def __str__(self):
        return f'User({self.id}, "{self.uuid}", "{self.username}", "{self.email}", {self.roles})'
