import {
  Token,
  loadToken,
  deleteToken,
  isExpired,
  refresh,
  loadRefreshToken,
  refreshExpiration,
  parseClaims,
  clearRefreshToken,
} from "@harrybrwn.com/api/auth";
import { SECOND } from "~/frontend/constants";

interface LoginManagerOptions {
  interval?: number;
  target?: EventTarget;
  clearToken?: () => void;
}

export default class LoginManager {
  private expirationCheckTimer: NodeJS.Timer;
  private refreshTokenTimeout: NodeJS.Timeout | null;
  private target: EventTarget;
  private loggedIn: boolean;
  private clearToken: () => void;

  private tokenChange<K extends keyof TokenChangeEventHandlersEventMap>(
    name: K,
    tok: Token | null
  ): TokenChangeEvent {
    return new CustomEvent(name, {
      detail: {
        signedIn: tok != null,
        token: tok,
        action: tok == null ? "logout" : "login",
      },
    });
  }

  private clearRefreshTimeout() {
    if (this.refreshTokenTimeout != null) {
      clearTimeout(this.refreshTokenTimeout);
      this.refreshTokenTimeout = null;
    }
  }

  private async refresh(): Promise<Token | null> {
    let refreshToken = loadRefreshToken();
    if (refreshToken == null) {
      return new Promise<Token | null>((_resolve, reject) => {
        reject(new Error("could not get refresh token from localStorage"));
      });
    }
    if (refreshToken.length == 0) {
      return new Promise<Token | null>((_resolve, reject) => {
        reject(new Error("stored refresh token has zero length"));
      });
    }
    let claims = parseClaims(refreshToken);
    let d = new Date(claims.exp * 1000);
    if (Date.now() < d.getTime()) {
      return new Promise<Token | null>((_resolve, reject) => {
        reject(new Error("refresh token is expired"));
      });
    }

    return refresh(refreshToken)
      .then((tok: Token) => {
        this.dispatch(tok);
        this.loggedIn = true;
        return tok;
      })
      .catch((error) => {
        // Could not refresh the access token
        console.error("refresh error:", error);
        this.dispatch(null);
        this.loggedIn = false;
        this.clearToken();
        // clearRefreshToken();
        return null;
      });
  }

  private doTimeout(token: Token) {
    let expires = new Date(token.expires * 1000);
    let now = new Date();
    let ms = expires.getTime() - now.getTime() - SECOND;
    if (ms < 0) {
      return;
    }
    this.clearRefreshTimeout();
    this.refreshTokenTimeout = setTimeout(() => {
      this.refresh()
        .then((token: Token | null) => {
          if (token != null) {
            this.doTimeout(token);
          }
        })
        .catch((error) => {
          console.error("could not refresh access token:", error);
        });
    }, ms);
  }

  constructor(options?: LoginManagerOptions) {
    if (!options) options = {};
    this.loggedIn = false;
    this.target = options.target || document;
    this.refreshTokenTimeout = null;
    this.clearToken = options.clearToken || deleteToken;

    // load the token and check expiration on startup
    let token = loadToken();
    if (token != null) {
      if (!isExpired(token)) {
        this.login(token);
      } else {
        let refExp = refreshExpiration();
        if (refExp == null || refExp.getTime() < Date.now()) {
          this.clearToken();
        } else {
          this.refresh().catch((error) => {
            console.error("unable to refresh access token:", error);
          });
        }
      }
    }

    // We only want to allocate a timer, so create one and clear right after.
    this.expirationCheckTimer = setInterval(() => {}, 120 * SECOND);
    clearInterval(this.expirationCheckTimer);

    // window.addEventListener("beforeunload", (ev: BeforeUnloadEvent) => {});
    this.target.addEventListener("visibilitychange", (ev: Event) => {
      if (!document.hidden) {
        //this.checkToken();
      }
    });
  }

  /**
   * Control login event dispatching.
   * @param tk a Token or null. null for logout, a token for login
   */
  private dispatch(tk: Token | null) {
    this.target.dispatchEvent(this.tokenChange("tokenChange", tk));
    this.target.dispatchEvent(this.tokenChange("loggedIn", tk));
  }

  logout() {
    this.dispatch(null);
    this.clearToken();
    this.loggedIn = false;
    this.clearRefreshTimeout();
  }

  login(tk: Token) {
    if (tk == null || isExpired(tk)) {
      throw new Error("cannot login with an invalid token");
    }
    this.dispatch(tk);
    this.loggedIn = true;
    this.doTimeout(tk);
  }

  stop() {
    clearInterval(this.expirationCheckTimer);
    this.clearRefreshTimeout();
  }

  isLoggedIn(): boolean {
    let token = loadToken();
    return token != null && this.loggedIn;
  }
}
