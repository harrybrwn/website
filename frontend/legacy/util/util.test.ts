import { storeToken } from "~/frontend/api/auth";
import { clearCookie } from "./cookies";
import { isEmail } from "./email";
import LoginManager from "./LoginManager";

describe("cookies", () => {
  test("clear cookies by name", () => {
    let name = "key";
    let d = new Date(Date.now() + 100000000);
    document.cookie = `${name}=value;expires=${d};path=/`;
    expect(document.cookie).toBe(`${name}=value`);
    clearCookie(name);
    expect(document.cookie).toEqual("");
  });
});

describe("email validation", () => {
  let valid = [
    "sindresorhus@gmail.com",
    "foo@bar",
    "test@about.museum",
    "test@nominet.org.uk",
    "test.test@sindresorhus.com",
    "test@255.255.255.255",
    "a@sindresorhus.com",
    "test@e.com",
    "test@xn--hxajbheg2az3al.xn--jxalpdlp",
    "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghiklm@sindresorhus.com",
    "test@g--a.com",
    "a@abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghikl.abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghikl.abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghikl.abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefg.hij",
    "123@sindresorhus.com",
    "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghiklmn@sindresorhus.com",
    "test@iana.co-uk",
    "a@a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v",
    "test@foo-bar.com",
    "foo@x.solutions",
  ];
  let invalid = [
    "@",
    "@io",
    "@sindresorhus.com",
    "test..sindresorhus.com",
    "test@iana..com",
    "test@sindresorhus.com.",
    "sindre@sindre@sindre.com",
    "mailto:sindresorhus@gmail.com",
    "foo.example.com",
    //
    "!#$%&amp;`*+/=?^`{|}~@sindresorhus.com",
    '"\\a"@sindresorhus.com',
    '""@sindresorhus.com',
    '"test"@sindresorhus.com',
    '"\\""@sindresorhus.com',
  ];
  test("valid emails", () => {
    for (const email of valid) {
      expect(isEmail(email)).toBeTruthy();
    }
  });
  test("invalid emails", () => {
    for (const email of invalid) {
      expect(isEmail(email)).toBeFalsy();
    }
  });
});

describe("login manager", () => {
  let m: LoginManager;

  beforeEach(() => {
    m = new LoginManager();
  });

  afterEach(() => {
    m.stop();
  });

  test("not logged in by default", () => {
    expect(m.isLoggedIn()).toBe(false);
  });

  test("login", () => {
    const refreshToken =
      "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsicmVmcmVzaCJdLCJleHAiOjE2NDM2OTA4OTcsImlhdCI6MTY0MzI1ODg5N30.bXMKEsa4Rji5d6KhFmu6U77Ww8MpadMGh5n7vUYbJ6zxU93x-E8uuutzyZdhkH_qJgGmyL2BSof8oY0ea0h3Cw";
    const rawToken =
      "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsidXNlciJdLCJleHAiOjE2NDMyNTg5MjcsImlhdCI6MTY0MzI1ODg5N30.J2pePgCOz_gA8lmcIxoxDzv0B6xxBeyC6UgAGz0Z77ImURf48jA9crj8JscwbGapycfSLi8kXPX5YrsnbR9JDQ";
    let token = {
      type: "Bearer",
      token: rawToken,
      refresh: refreshToken,
      expires: Math.round((Date.now() + 100000) / 1000),
    };
    storeToken(token);
    expect(m.isLoggedIn()).toBe(false);

    let eventTriggered = false;
    document.addEventListener("tokenChange", (ev: TokenChangeEvent) => {
      eventTriggered = true;
      expect(ev.detail.action).toBe("login");
    });

    m.login(token);
    expect(m.isLoggedIn()).toBe(true);
    expect(eventTriggered).toBe(true);
  });
});
