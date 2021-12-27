import "./styles/font.css";
import "./styles/main.css";
import {
  TOKEN_KEY,
  Token,
  loadToken,
  isExpired,
  login,
  deleteToken,
  storeToken,
  setCookie,
} from "./api/auth";
import { clearCookie } from "./util";
import { ThemeManager } from "./components/theme";
import { Modal } from "./components/modal";
import "./components/toggle";
import * as api from "./api";

function handleLogin(formID: string, callback: (t: Token) => void) {
  let formOrNull = document.getElementById(formID) as HTMLFormElement | null;
  if (formOrNull == null) {
    throw new Error("could not find element " + formID);
  }
  let form: HTMLFormElement = formOrNull;
  form.addEventListener("submit", function (event: SubmitEvent) {
    event.preventDefault();
    let formData = new FormData(form);
    login({
      username: formData.get("username") as string,
      email: formData.get("email") as string,
      password: formData.get("password") as string,
    })
      .then((tok: Token) => {
        callback(tok);
        form.reset();
      })
      .catch((error: Error) => {
        console.error(error);
        let e = document.createElement("p");
        e.innerHTML = `${error}`;
        form.appendChild(e);
      });
  });
}

function handleLogout(id: string, callback: () => void) {
  let btn = document.getElementById(id);
  if (btn == null) {
    console.error("could not find logout button");
    return;
  }
  btn.addEventListener("click", (ev: MouseEvent) => {
    callback();
  });
}

const applyPageCount = () => {
  let countBox = document.getElementById("hit-count");
  if (countBox == null) {
    return;
  }
  let container = countBox;
  api.hits("/").then((hits) => {
    container.innerText = `page visits: ${hits.count}`;
  });
};

const SECOND = 1000;

const loginButtonID = "login-btn";
const loginPanelID = "login-panel";

class LoginPopup {
  loginBtn: HTMLElement;
  loginPanel: HTMLElement;
  open: boolean;

  constructor() {
    this.open = false;
    this.escCloser = (ev: KeyboardEvent) => {};
    this.clickCloser = (ev: MouseEvent) => {};
    this.loginBtn =
      document.getElementById(loginButtonID) ||
      document.createElement("button");
    this.loginPanel =
      document.getElementById(loginPanelID) || document.createElement("div");
  }

  private escCloser: (ev: KeyboardEvent) => void;
  private clickCloser: (ev: MouseEvent) => void;

  private _toggle() {
    this.loginPanel.style.display = this.open ? "none" : "block";
    this.open = !this.open;
  }

  private cleanup() {
    window.removeEventListener("click", this.clickCloser);
    window.removeEventListener("keydown", this.escCloser);
  }

  toggle() {
    if (!this.open) {
      this.clickCloser = (ev: MouseEvent) => {
        let el = ev.target as HTMLElement | null;
        if (el != null && el.id == "show-login-btn") return;
        while (el != null && el != document.body) {
          if (el == this.loginPanel || el.id == "show-login-btn") {
            return;
          }
          el = el.parentElement;
        }
        this._toggle();
        this.cleanup();
      };
      this.escCloser = (ev: KeyboardEvent) => {
        if (ev.key != "Escape") {
          return;
        }
        this._toggle();
        this.cleanup();
      };
    }

    this._toggle();
    if (this.open) {
      window.addEventListener("click", this.clickCloser);
      window.addEventListener("keydown", this.escCloser);
    } else {
      this.cleanup();
    }
  }

  listen() {
    this.loginBtn.addEventListener("click", () => {
      this.toggle();
    });
  }
}

interface LoginManagerOptions {
  // The interval at which the manager checks to see if we are expired
  // TODO use setTimeout on a login event to handle this so we don't have to poll
  interval?: number;

  target?: EventTarget;
}

class LoginManager {
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

const anchor = (href: string, text: string): HTMLAnchorElement => {
  let a = document.createElement("a");
  a.href = href;
  a.innerText = text;
  return a;
};

const privateLinks = (): HTMLLIElement[] => {
  let els = [
    document.createElement("li"),
    document.createElement("li"),
    document.createElement("li"),
  ];
  els[0].appendChild(anchor("/tanya/hyt", "tanya y harry"));
  els[1].appendChild(anchor("/old", "old site"));
  els[2].appendChild(anchor("./admin", "admin"));
  return els;
};

const main = () => {
  let themeManager = new ThemeManager();
  let loginManager = new LoginManager({ interval: 5 * 60 * SECOND });
  let loginPanel = new LoginPopup();
  let showLoginBtn = document.getElementById("show-login-btn");
  showLoginBtn?.addEventListener("click", (ev: MouseEvent) => {
    loginPanel.toggle();
  });

  themeManager.onChange((ev: Event) => {
    themeManager.toggle();
  });

  let helpWindow = new Modal({
    open: false,
    buttonID: "help-btn",
    modalID: "help-window",
  });
  document
    .getElementById("help-btn")
    ?.addEventListener("click", (ev: MouseEvent) => {
      helpWindow.toggle();
    });

  // Logged in stuff
  let links = document.querySelector(".links");
  if (!links) {
    console.error("could not find .links");
  }
  let tanya = document.createElement("a");
  tanya.href = "/tanya/hyt";
  tanya.className = "tanya-link";
  tanya.innerText = "tanya y harry";
  let li = document.createElement("li");
  li.appendChild(tanya);

  let privLinks = privateLinks();
  if (loginManager.isLoggedIn()) {
    // links?.appendChild(li);
    for (let li of privLinks) {
      links?.appendChild(li);
    }
  }

  // Handle login and logout
  document.addEventListener("tokenChange", (ev: TokenChangeEvent) => {
    const e = ev.detail;
    if (e.action == "login") {
      storeToken(e.token);
      setCookie(e.token);
      // links?.appendChild(li);
      for (let li of privLinks) {
        links?.appendChild(li);
      }
    } else {
      clearCookie(TOKEN_KEY);
      deleteToken();
      // links?.removeChild(li);
      for (let li of privLinks) {
        links?.removeChild(li);
      }
    }
  });

  loginPanel.listen();
  handleLogout("logout-btn", () => {
    loginManager.logout();
  });
  handleLogin("login-form", (tok: Token) => {
    loginManager.login(tok);
    loginPanel.toggle();
  });

  // Close login window when the minimize or close buttons are pressed
  for (let id of ["login-window-close", "login-window-minimize"]) {
    document.getElementById(id)?.addEventListener("click", (ev: MouseEvent) => {
      if (loginPanel.open) loginPanel.toggle();
    });
  }
  for (let id of ["help-window-close", "help-window-minimize"]) {
    document.getElementById(id)?.addEventListener("click", (ev: MouseEvent) => {
      if (loginPanel.open) helpWindow.toggle();
    });
  }

  document.addEventListener("keydown", (ev: KeyboardEvent) => {
    const e = ev.target as HTMLElement;
    if (e.tagName == "INPUT" || e.tagName == "TEXTAREA") {
      return;
    }
    switch (ev.key) {
      case "l":
        ev.preventDefault();
        loginPanel.toggle();
        break;
      case "t":
        themeManager.toggle();
        themeManager.themeToggle.checked = !themeManager.themeToggle.checked;
        break;
      case "?":
        helpWindow.toggle();
        break;
    }
  });

  let welcomeTicker = 0;
  let welcomeBanner = document.getElementsByClassName(
    "welcome-banner"
  )[0] as HTMLElement;
  let colors = ["red", "blue", "mediumspringgreen", "purple", "pink", "yellow"];
  setInterval(() => {
    welcomeBanner.style.color = colors[welcomeTicker % colors.length];
    welcomeTicker++;
  }, 500);

  applyPageCount();
};

main();
