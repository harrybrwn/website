import { Token, loadToken, isExpired } from "~/frontend/api/auth";
import { SECOND } from "~/frontend/constants";

// const SECOND = 1000;

interface LoginManagerOptions {
  // The interval at which the manager checks to see if we are expired
  // TODO use setTimeout on a login event to handle this so we don't have to poll
  interval?: number;

  target?: EventTarget;
}

export default class LoginManager {
  private expirationCheckTimer: NodeJS.Timer;
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

  constructor(options: LoginManagerOptions) {
    this.target = options.target || document;
    // load the token and check expiration on startup
    let token = loadToken();
    if (token != null && !isExpired(token)) {
      this.login(token);
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
  }

  login(tk: Token) {
    this.target.dispatchEvent(this.tokenChange("tokenChange", tk));
    this.target.dispatchEvent(this.tokenChange("loggedIn", tk));
  }

  stop() {
    clearInterval(this.expirationCheckTimer);
  }

  isLoggedIn(): boolean {
    let token = loadToken();
    return token != null;
  }
}
