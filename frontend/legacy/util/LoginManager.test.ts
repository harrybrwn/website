import {
  TOKEN_KEY,
  Token,
  Role,
  clearRefreshToken,
  deleteToken,
  storeToken,
} from "~/frontend/api/auth";
import { clearCookie } from "./cookies";
import LoginManager from "./LoginManager";
import MockFetch from "./MockFetch";
import { newToken } from "./mocks";
import { HOUR } from "~/frontend/constants";

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const testToken = (offset: number, refresh_exp?: number): Token => {
  return newToken(
    {
      id: Math.random() % 100,
      uuid: "",
      roles: [Role.Admin],
      aud: ["unit-test-user"],
      iss: "harrybrwn.com-tests",
      exp: Math.round((Date.now() + offset) / 1000),
      iat: 0,
    },
    refresh_exp
  );
};

describe("new LoginManager", () => {
  describe("when logged in", () => {
    afterEach(() => {
      deleteToken();
    });

    test("token expired", () => {
      let tok = testToken(-10_000, -1000);
      storeToken(tok);
      let m = new LoginManager();
      expect(m.isLoggedIn()).toBe(false);
    });

    test("token ok", () => {
      let tok = testToken(1000);
      storeToken(tok);
      let m = new LoginManager();
      expect(m.isLoggedIn()).toBe(true);
    });
  });

  describe("logged out", () => {
    afterEach(() => {
      localStorage.clear();
      clearCookie(TOKEN_KEY);
    });

    describe("no refresh token", () => {
      let m: LoginManager;

      beforeEach(() => {
        clearRefreshToken();
        deleteToken();
        m = new LoginManager();
      });

      test("logged out", () => {
        expect(m.isLoggedIn()).toBe(false);
      });
    });

    describe("has refresh token", () => {
      let mockFetch: MockFetch;
      let token: Token;

      beforeEach(() => {
        mockFetch = new MockFetch();
        mockFetch.start();
        token = newToken(
          {
            id: 1,
            uuid: "",
            roles: [Role.Admin],
            aud: ["user", "yee", "yee"],
            iss: "",
            // exp: Math.round((Date.now() - 24 * HOUR) / 1000),
            // exp: Math.round(Date.now() / 1000),
            exp: Math.round(Date.now() / 1000) - 24 * 60 * 60 * 2,
            iat: 0,
          },
          // Date.now() - 60 * 60 * 1000
          // Date.now() - HOUR
          Math.round(Date.now() / 1000) + 24 * 60 * 60 * 2
        );
        storeToken(token);
      });

      afterEach(() => {
        mockFetch.finish();
      });

      test("rotates access token", async () => {
        mockFetch
          .expect("/api/refresh?cookie=true", {
            method: "POST",
            headers: {
              Accept: "application/json",
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ refresh_token: token.refresh }),
          })
          .returns({
            ok: true,
            json: () => {
              let t = testToken(HOUR, HOUR * 2);
              storeToken(t);
              return Promise.resolve(t);
            },
          } as Response);

        await sleep(500);
        let m = new LoginManager();
        await sleep(500);
        expect(m.isLoggedIn()).toBe(true);
      });
      //
    });
  });
});
