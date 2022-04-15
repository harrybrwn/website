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
            exp: Math.round((Date.now() - 24 * HOUR) / 1000),
            // exp: Math.round(Date.now() / 1000) - 24 * 60 * 60,
            iat: 0,
          },
          // Date.now() - 60 * 60 * 1000
          Date.now() - HOUR
        );
        storeToken(token);
      });

      afterEach(() => {
        mockFetch.finish();
      });

      test("rotates auth token", () => {
        mockFetch
          .expect("/api/refresh?cookie=true", {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Accept: "application/json",
            },
            body: JSON.stringify({ refresh_token: token.refresh }),
          })
          .returns({
            ok: true,
            json: () => Promise.resolve(testToken(HOUR, HOUR * 2)),
          } as Response);
        let m = new LoginManager();
        // expect(m.isLoggedIn()).toBe(true);
        // console.log("loggedIn:", m.isLoggedIn());
      });
      //
    });
  });
});
