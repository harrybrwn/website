import {
  parseClaims,
  login,
  loadRefreshToken,
  loadToken,
  clearRefreshToken,
  deleteToken,
  refresh,
  setCookie,
} from "./auth";
import MockFetch from "~/frontend/util/MockFetch";
import { toBindingIdentifierName } from "@babel/types";

test("parse jwt claims", () => {
  let token =
    "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsicmVmcmVzaCJdLCJleHAiOjE2NDM1MzEwMTQsImlhdCI6MTY0MzA5OTAxNH0.DnesQeMrNeBnDnW0ROKe28mGvDxeoiph4YOjcUjRTkC1J0sf7wVTH36deuJfPbiD-MBFh1aPHMI0jX7djw76Ag";
  let claims = parseClaims(token);
  expect(claims.id).not.toBeLessThan(1);
  expect(claims.uuid).not.toHaveLength(0);
  expect(claims.uuid).toEqual("5743e8f5-4d6f-49a0-80fc-d3201d41abe7");
  expect(claims.iss).toEqual("harrybrwn.com");
  expect(claims.roles).toHaveLength(1);
  expect(claims.aud).toHaveLength(1);
  expect(claims.roles[0]).toEqual("admin");
  expect(claims.aud[0]).toEqual("refresh");
});

describe("login", () => {
  const refreshToken =
    "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsicmVmcmVzaCJdLCJleHAiOjE2NDM2OTA4OTcsImlhdCI6MTY0MzI1ODg5N30.bXMKEsa4Rji5d6KhFmu6U77Ww8MpadMGh5n7vUYbJ6zxU93x-E8uuutzyZdhkH_qJgGmyL2BSof8oY0ea0h3Cw";
  const rawToken =
    "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsidXNlciJdLCJleHAiOjE2NDMyNTg5MjcsImlhdCI6MTY0MzI1ODg5N30.J2pePgCOz_gA8lmcIxoxDzv0B6xxBeyC6UgAGz0Z77ImURf48jA9crj8JscwbGapycfSLi8kXPX5YrsnbR9JDQ";
  const user = {
    username: "tester",
    email: "tester@example.com",
    password: "password1",
  };

  let mockFetch: MockFetch;

  beforeEach(() => {
    mockFetch = new MockFetch();
    mockFetch.start();
  });

  afterEach(() => {
    mockFetch.finish();
    localStorage.clear();
  });

  describe("login successful", () => {
    beforeEach(() => {
      mockFetch
        .expect("/api/token?cookie=true", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(user),
        })
        .returns({
          ok: true,
          json() {
            return Promise.resolve({
              refresh_token: refreshToken,
              token: rawToken,
              token_type: "Bearer",
              expires: 1643258927,
            });
          },
        } as Response);
    });

    test("token matches", async () => {
      let token = await login(user);
      expect(fetch).toHaveBeenCalledTimes(1);
      expect(token.token).toBe(rawToken);
      expect(token.refresh).toBe(refreshToken);
      expect(token.expires).toBe(1643258927);
      expect(token.type).toBe("Bearer");
    });

    test("refresh token set", async () => {
      let token = await login(user);
      let refTok = loadRefreshToken();
      expect(fetch).toHaveBeenCalledTimes(1);
      expect(refTok).not.toBe(null);
      expect(refTok).toBe(refreshToken);
      expect(refTok).toBe(token.refresh);
      clearRefreshToken();
      refTok = loadRefreshToken();
      expect(refTok).toBe(null);
      expect(refTok).not.toBe(refreshToken);
      let t = loadToken();
      if (t == null) fail("load token should not be null");
      expect(t).not.toBe(null);
      expect(t.token).toEqual(token.token);
      expect(t.expires).toEqual(token.expires);
      expect(t.type).toEqual(token.type);
      deleteToken();
    });
  });

  describe("login failed", () => {
    beforeEach(() => {
      mockFetch
        .expect("/api/token?cookie=true", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(user),
        })
        .returns({
          ok: false,
          json() {
            return Promise.resolve({ message: "error message" });
          },
        } as Response);
    });

    test("throws an error on bad response", async () => {
      try {
        await login(user);
      } catch (error) {
        expect(fetch).toHaveBeenCalledTimes(1);
        if (error instanceof Error) {
          expect(error).toBeTruthy();
          expect(error.message).toBe("error message");
        } else {
          fail("expected error of type Error");
        }
        expect(loadToken()).toBe(null);
      } finally {
        deleteToken();
      }
    });

    test("does not save tokens", async () => {
      try {
        await login(user);
      } catch (error) {}
      expect(fetch).toHaveBeenCalledTimes(1);
      expect(loadRefreshToken()).toBe(null);
      expect(loadToken()).toBe(null);
    });
  });
});

describe("refresh", () => {
  const refreshToken =
    "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsicmVmcmVzaCJdLCJleHAiOjE2NDM2OTA4OTcsImlhdCI6MTY0MzI1ODg5N30.bXMKEsa4Rji5d6KhFmu6U77Ww8MpadMGh5n7vUYbJ6zxU93x-E8uuutzyZdhkH_qJgGmyL2BSof8oY0ea0h3Cw";
  const rawToken =
    "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsidXNlciJdLCJleHAiOjE2NDMyNTg5MjcsImlhdCI6MTY0MzI1ODg5N30.J2pePgCOz_gA8lmcIxoxDzv0B6xxBeyC6UgAGz0Z77ImURf48jA9crj8JscwbGapycfSLi8kXPX5YrsnbR9JDQ";
  const user = {
    username: "tester",
    email: "tester@example.com",
    password: "password1",
  };

  const oldToken = {
    token:
      "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwid…IRZ9oA2pGCN-3gzCIQCBNffJbCcoviiniih8G5B2RutbjPjCw",
    refresh:
      "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsicmVmcmVzaCJdLCJleHAiOjE2NDM3NzU0MzcsImlhdCI6MTY0MzM0MzQzN30._FEJKd3ixVSzeePMm1VT-dSwg4YNuC37i29oaHhzorKUO3VRaaFiT1in7RyMsL0EDD1QZqcb6PFffIexTKi5Dg",
    expires: 1643343496,
    type: "Bearer",
  };

  let mockFetch: MockFetch;
  let newExpiration: number;

  beforeEach(() => {
    mockFetch = new MockFetch();
    mockFetch.start();
  });

  afterEach(() => {
    mockFetch.finish();
    localStorage.clear();
  });

  describe("successful refresh", () => {
    newExpiration = Math.round((Date.now() + 100000) / 1000);
    beforeEach(() => {
      mockFetch
        .expect("/api/refresh?cookie=true", {
          method: "POST",
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ refresh_token: oldToken.refresh }),
        })
        .returns({
          ok: true,
          json() {
            return Promise.resolve({
              refresh_token: refreshToken,
              token: rawToken,
              token_type: "Bearer",
              expires: newExpiration,
            });
          },
        } as Response);
    });

    test("refresh ok", async () => {
      let token = await refresh(oldToken.refresh);
      expect(token.refresh).toBe(refreshToken);
      expect(token.token).toBe(rawToken);
      expect(token.expires).toBe(newExpiration);
      expect(token.type).toBe("Bearer");
    });
  });

  describe("failed refresh", () => {
    beforeEach(() => {
      mockFetch
        .expect("/api/refresh?cookie=true", {
          method: "POST",
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ refresh_token: oldToken.refresh }),
        })
        .returns({ ok: false, status: 500 } as Response);
    });

    test("should throw an error", async () => {
      try {
        await refresh(oldToken.refresh);
        fail("should throw error");
      } catch (error) {
        expect(fetch).toHaveBeenCalledTimes(1);
        if (error instanceof Error) {
          expect(error).toBeTruthy();
          expect(error.message).toBe(
            "could not get new access token: status 500"
          );
        } else {
          fail("expected error of type Error");
        }
      }
    });
  });
});

describe("setCookie", () => {
  const token = {
    token:
      "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwid…IRZ9oA2pGCN-3gzCIQCBNffJbCcoviiniih8G5B2RutbjPjCw",
    refresh:
      "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6IjU3NDNlOGY1LTRkNmYtNDlhMC04MGZjLWQzMjAxZDQxYWJlNyIsInJvbGVzIjpbImFkbWluIl0sImlzcyI6ImhhcnJ5YnJ3bi5jb20iLCJhdWQiOlsicmVmcmVzaCJdLCJleHAiOjE2NDM3NzU0MzcsImlhdCI6MTY0MzM0MzQzN30._FEJKd3ixVSzeePMm1VT-dSwg4YNuC37i29oaHhzorKUO3VRaaFiT1in7RyMsL0EDD1QZqcb6PFffIexTKi5Dg",
    expires: 1643343496,
    type: "Bearer",
  };
  setCookie(token);
  test("should have token cookie", () => {
    expect(document.cookie).toBe("_token=" + token.token);
  });
});
