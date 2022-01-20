import { Token, loadToken, isExpired, refresh } from "~/frontend/api/auth";
import { SECOND } from "~/frontend/constants";

interface LoginManagerOptions {
  // The interval at which the manager checks to see if we are expired
  // TODO use setTimeout on a login event to handle this so we don't have to poll
  interval?: number;

  target?: EventTarget;
}

export default class LoginManager {
  private expirationCheckTimer: NodeJS.Timer;
  private refreshTokenTimeout: NodeJS.Timeout | null;
  private target: EventTarget;

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

  private doTimeout(token: Token) {
    let expires = new Date(token.expires * 1000);
    let now = new Date();
    let ms = expires.getTime() - now.getTime() - SECOND;
    console.log("token expires at", expires);
    console.log("expires in", ms, "milliseconds");
    if (ms < 0) {
      return;
    }
    if (this.refreshTokenTimeout != null) {
      clearTimeout(this.refreshTokenTimeout);
    }
    this.refreshTokenTimeout = setTimeout(() => {
      console.log("refreshing jwt token");
      refresh().then((tok: Token) => {
        this.doTimeout(tok);
      });
    }, ms);
  }

  constructor(options: LoginManagerOptions) {
    this.target = options.target || document;
    this.refreshTokenTimeout = setTimeout(() => {}, 0);
    // load the token and check expiration on startup
    let token = loadToken();
    console.log(token, token?.expires);
    if (token != null && !isExpired(token)) {
      console.log("token not expired");
      this.login(token);
      this.doTimeout(token);
    } else {
      console.log("token expired");
    }
    this.expirationCheckTimer = setInterval(() => {
      let token = loadToken();
      if (token == null) {
        this.logout();
        return;
      }
      if (isExpired(token)) {
        this.logout();
      } else {
        console.log("token still valid");
      }
    }, options.interval || 60 * SECOND);
  }

  logout() {
    this.target.dispatchEvent(this.tokenChange("tokenChange", null));
    this.target.dispatchEvent(this.tokenChange("loggedIn", null));
    if (this.refreshTokenTimeout != null) {
      clearTimeout(this.refreshTokenTimeout);
      this.refreshTokenTimeout = null;
    }
  }

  login(tk: Token) {
    this.target.dispatchEvent(this.tokenChange("tokenChange", tk));
    this.target.dispatchEvent(this.tokenChange("loggedIn", tk));
    if (this.refreshTokenTimeout != null) {
      clearTimeout(this.refreshTokenTimeout);
      this.refreshTokenTimeout = null;
    }
    this.doTimeout(tk);
  }

  stop() {
    clearInterval(this.expirationCheckTimer);
  }

  isLoggedIn(): boolean {
    let token = loadToken();
    return token != null;
  }
}
