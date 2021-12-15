import {
  TOKEN_KEY,
  Token,
  loadToken,
  isExpired,
  login,
  deleteToken,
} from "./auth";
import { clearCookie } from "./util";

function handleLogin(formID: string, callback: (t: Token) => void) {
  let form = document.getElementById(formID) as HTMLFormElement | null;
  if (form == null) {
    throw new Error("could not find element " + formID);
  }
  form.addEventListener("submit", function (event: SubmitEvent) {
    event.preventDefault();
    if (form == null) {
      throw new Error("could not find form element");
    }
    let formData = new FormData(form);
    login({
      username: formData.get("username") as string,
      email: formData.get("email") as string,
      password: formData.get("password") as string,
    })
      .then(callback)
      .catch(console.error);
  });
}

function handleLogout(id: string) {
  let btn = document.getElementById(id);
  if (btn == null) {
    console.error("could not find logout button");
    return;
  }
  btn.addEventListener("click", (ev: MouseEvent) => {
    clearCookie(TOKEN_KEY);
    deleteToken();
    console.log("tokens cleared");
  });
}

const onExpired = (token: Token): boolean => {
  if (token == null) {
    return false;
  }
  if (isExpired(token)) {
    // TODO get new token using refresh token
    console.log("token is expired");
    return false;
  } else {
    console.log("token still valid");
    return true;
  }
};

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;

function handleLoginPopup() {
  let loginBtn = document.getElementById("login-btn");
  let loginPanel = document.getElementById("login-panel");
  if (loginBtn == null) {
    throw new Error("could not find login button");
  }

  let open = false;
  loginBtn.addEventListener("click", (event: MouseEvent) => {
    if (loginPanel == null) {
      throw new Error("could not find login panel");
    }
    if (open) {
      loginPanel.style.display = "none";
      open = false;
    } else {
      loginPanel.style.display = "block";
      open = true;
    }
  });
}

const main = () => {
  let signedIn = false;
  let refreshTokenTimer: NodeJS.Timer;
  let token: Token | null;
  handleLogin("login-form", (tok: Token) => {
    signedIn = true;
    token = tok;
  });
  handleLogout("logout-btn");

  token = loadToken();
  if (token == null) {
    console.error("could not load token");
  } else {
    if (!isExpired(token)) {
      signedIn = true;
      onExpired(token);
    }
  }

  refreshTokenTimer = setInterval(() => {
    let token = loadToken();
    if (token == null) return;
    onExpired(token);
  }, 30 * SECOND);
  //clearInterval(refreshTokenTimer);
  handleLoginPopup();

  // document.addEventListener("keydown", (ev: KeyboardEvent) => {
  //   const e = ev.target as HTMLElement;
  //   console.log(e, e.tagName);
  //   if (e.tagName == "INPUT" || e.tagName == "TEXTAREA") {
  //     return;
  //   }
  //   ev.preventDefault();
  //   console.log(ev);
  // });
};

main();
