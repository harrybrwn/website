import {
  Token,
  loadToken,
  deleteToken,
  isExpired,
  refresh,
  loadRefreshToken,
  clearRefreshToken,
} from "~/frontend/api/auth";
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
    console.log("refreshing jwt token");
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
    return refresh(refreshToken)
      .then((tok: Token) => {
        this.dispatch(tok);
        this.loggedIn = true;
        return tok;
      })
      .catch((error) => {
        // Could not refresh the access token
        console.log("refresh error:", error);
        this.dispatch(null);
        this.loggedIn = false;
        this.clearToken();
        clearRefreshToken();
        return null;
      });
  }

  private doTimeout(token: Token) {
    let expires = new Date(token.expires * 1000);
    let now = new Date();
    let ms = expires.getTime() - now.getTime() - SECOND;
    console.log("expires in", ms, "milliseconds at", expires);
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
          console.log("could not refresh access token:", error);
        });
    }, ms);
  }

  private async checkToken() {
    let token = loadToken();
    if (token == null) {
      console.log("TokenManager: no token found");
      if (loadRefreshToken() != null) {
        await this.refresh();
      }
    } else if (isExpired(token)) {
      console.log("TokenManager: token expired");
      await this.refresh();
    } else {
      console.log("token still valid");
    }
  }

  constructor(options: LoginManagerOptions) {
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
        this.clearToken();
      }
    }

    this.expirationCheckTimer = setInterval(() => {}, 120 * SECOND);
    clearInterval(this.expirationCheckTimer);

    document.addEventListener("visibilitychange", (ev: Event) => {
      if (!document.hidden) {
        this.checkToken();
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
