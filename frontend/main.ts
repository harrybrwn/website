import "./styles/main.css";
import {
  TOKEN_KEY,
  Token,
  login,
  deleteToken,
  storeToken,
  setCookie,
  clearRefreshToken,
} from "./api/auth";
import { SECOND } from "./constants";
import { clearCookie } from "./util/cookies";
import LoginManager from "./util/login_manager";
import { ThemeManager } from "./components/theme";
import { Modal } from "./components/modal";
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
        return tok;
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

const focusOnLoginEmail = () => {
  let email = document.querySelector(
    "#login-form input[type=email]"
  ) as HTMLInputElement;
  if (email) {
    email.focus();
  }
};

const welcomeBannerColors = (banner: HTMLElement | null, ms: number) => {
  if (banner == null) {
    return;
  }
  let welcomeTicker = 0;
  let colors = [
    "red",
    "orange",
    "yellow",
    "mediumspringgreen",
    "blue",
    "purple",
    "pink",
  ];
  const fn = () => {
    banner.style.color = colors[welcomeTicker % colors.length];
    welcomeTicker++;
  };
  setInterval(fn, ms);
};

const webButtonClipboard = (num: number) => {
  let tooltip = document.getElementById(`web-btn-${num}-tooltip`);
  if (tooltip == null) {
    console.error("could not find tooltip");
    return;
  }
  if (tooltip.children.length == 0 || tooltip.children[0].tagName != "IMG") {
    throw new Error("web button tooltip has no child image");
  }
  let button = tooltip.children[0] as HTMLImageElement;

  const defaultMsg = "Copy code";
  const payload = `<a href="${window.origin}/">\n  <img src="${button.src}" alt="Harry Brown" width="88" height="31">\n</a>`;

  tooltip.setAttribute("data-text", defaultMsg);
  const copy = () => {
    navigator.clipboard.writeText(payload);
    tooltip?.setAttribute("data-text", "Code copied!");
    fetch(`/api/ping?action=web-button-copy&button=${button.src}`);
  };
  button.addEventListener("click", copy);
  // tooltip.addEventListener("click", copy);
  button.addEventListener("mouseout", () => {
    tooltip?.setAttribute("data-text", defaultMsg);
  });
};

const main = () => {
  let themeManager = new ThemeManager();
  let loginManager = new LoginManager({
    interval: 5 * 60 * SECOND,
    clearToken: () => {
      deleteToken();
      clearCookie(TOKEN_KEY);
    },
  });
  let loginPanel = new Modal({
    button: document.getElementById("login-btn"),
    element: document.getElementById("login-panel"),
  });
  let helpWindow = new Modal({
    button: document.getElementById("help-btn"),
    element: document.getElementById("help-window"),
  });
  // Another login panel button
  document
    .getElementById("show-login-btn")
    ?.addEventListener("click", (ev: MouseEvent) => {
      loginPanel.toggle();
      if (loginPanel.open) focusOnLoginEmail();
    });
  // Toggle help window button
  helpWindow.toggleOnClick();
  // Handle theme changes
  themeManager.onChange((_: Event) => {
    themeManager.toggle();
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
    for (let li of privLinks) {
      links?.appendChild(li);
    }
  }

  // Handle login and logout
  document.addEventListener("tokenChange", (ev: TokenChangeEvent) => {
    const e = ev.detail;
    console.log("token change:", e.action);
    if (e.action == "login") {
      storeToken(e.token);
      setCookie(e.token);
      for (let li of privLinks) {
        links?.appendChild(li);
      }
    } else {
      if (!loginManager.isLoggedIn()) {
        return;
      }
      for (let li of privLinks) {
        links?.removeChild(li);
      }
    }
  });

  loginPanel.toggleOnClick();
  handleLogout("logout-btn", () => {
    loginManager.logout();
    // TODO send revoke to token service
    clearRefreshToken();
  });
  handleLogin("login-form", (tok: Token) => {
    loginManager.login(tok);
    loginPanel.toggle();
  });

  // Close login window when the minimize or close buttons are pressed
  for (let id of ["login-window-close", "login-window-minimize"]) {
    document.getElementById(id)?.addEventListener("click", (_: MouseEvent) => {
      if (loginPanel.open) loginPanel.toggle();
    });
  }
  // Close and minimize buttons for help window
  for (let id of ["help-window-close", "help-window-minimize"]) {
    document.getElementById(id)?.addEventListener("click", (_: MouseEvent) => {
      if (helpWindow.open) helpWindow.toggle();
    });
  }

  // Handle Keypresses
  document.addEventListener("keydown", (ev: KeyboardEvent) => {
    const e = ev.target as HTMLElement;
    if (e.tagName == "INPUT" || e.tagName == "TEXTAREA") {
      return;
    }
    switch (ev.key) {
      case "l":
        ev.preventDefault();
        loginPanel.toggle();
        if (loginPanel.open) focusOnLoginEmail();
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

  welcomeBannerColors(document.querySelector(".welcome-banner"), SECOND);
  applyPageCount();
  webButtonClipboard(1);
  webButtonClipboard(2);
};

main();
