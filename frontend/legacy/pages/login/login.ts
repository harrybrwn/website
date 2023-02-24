import "~/frontend/styles/font.css";
import { isEmail } from "~/frontend/util/email";
import { applyTheme } from "~/frontend/components/theme";
import { new_pkce, load_pkce, delete_pkce, save_pkce } from "./pkce";
import { OAuth2Token, RedirectTarget, login, consent } from "./oidc";
import { getCookie } from "~/frontend/util/cookies";

const main = () => {
  try {
    applyTheme();
  } catch (err) {}
  let form = document.getElementById("login-form") as HTMLFormElement;
  if (form == null) {
    throw new Error("could not find login form");
  }
  const loading = document.getElementById("loading-box");
  if (loading === null) {
    let e = new Error("failed to grab loading screen");
    handleError(e);
    throw e;
  }
  const credsBox = document.getElementById("creds-box");
  if (!credsBox) {
    throw new Error("could not get credentials box");
  }
  const oidcbtn = document.getElementById("with-oidc");
  if (oidcbtn !== null) {
    oidcbtn.addEventListener("click", async (ev: MouseEvent) => {
      let pkce = await new_pkce();
      delete_pkce();
      save_pkce(pkce);

      let u = new URL("/oauth2/auth", OIDC_URL);
      u.searchParams.set("client_id", OIDC_CLIENT_ID);
      u.searchParams.set("response_type", "code");
      u.searchParams.set("scope", "openid offline");
      u.searchParams.set("state", pkce.state);
      u.searchParams.set("code_challenge", pkce.challenge);
      u.searchParams.set("code_challenge_method", "S256");
      window.location.href = u.toString();
    });
  }

  const params = new URLSearchParams(window.location.search);
  const code = params.get("code");
  const consent_challenge = params.get("consent_challenge");
  const login_challenge = params.get("login_challenge");
  const forceLogin = params.get("force_login");
  let authToken = getCookie("_token");

  // If we are already logged in and going through the oidc flow then we should
  // hide the username/password form and display the loading panel.
  if (authToken !== null && (login_challenge || consent_challenge)) {
    loading.style.visibility = "visible";
    credsBox.style.visibility = "hidden";
  }

  if (oidcbtn && (login_challenge || consent_challenge || code)) {
    oidcbtn.style.visibility = "hidden";
  }

  if (consent_challenge) {
    consent(consent_challenge)
      .then((res: RedirectTarget) => {
        window.location.href = res.redirect_to;
      })
      .catch((err: Error) => {
        handleError(err, form);
        form.reset();
      });
    return;
  } else if (code) {
    // if is mainly just here for testing...
    oidcToken(code)
      .then(async (token) => {
        let node = document.createElement("p");
        node.innerText = JSON.stringify(token, null, 4);
        document.body.appendChild(node);
        //window.location.pathname = "/";
        let res = await fetch(new URL("/userinfo", OIDC_URL).toString(), {
          method: "GET",
          headers: {
            Accept: "application/json",
            Authorization: `Bearer ${token.access_token}`,
          },
        });
        let info = await res.json();
        console.log("userinfo:", info);
      })
      .catch((e) => handleError(e, form));
    return;
  }

  // Auto login if auth cookie is found.
  if (!forceLogin && authToken !== null && login_challenge) {
    login({ login_challenge: login_challenge })
      .then(async (blob: RedirectTarget) => {
        if (!blob.redirect_to) {
          throw new Error("to redirect target");
        }
        await sleep(1000);
        window.location.href = blob.redirect_to;
      })
      .catch((err: Error) => {
        handleError(err, form);
        form.reset();
      });
  }

  form.addEventListener("submit", (ev: SubmitEvent) => {
    ev.preventDefault();
    let d = new FormData(ev.target as HTMLFormElement);
    let ident = d.get("identifier") as string;
    let req = {
      password: d.get("password") as string,
      email: "",
      username: "",
      login_challenge: login_challenge,
    };
    if (isEmail(ident)) {
      req.email = ident;
    } else {
      req.username = ident;
    }
    login(req)
      .then((blob: RedirectTarget) => {
        if (blob.redirect_to) {
          window.location.href = blob.redirect_to;
        } else {
          window.location.pathname = "/";
        }
      })
      .catch((err: Error) => {
        handleError(err, form);
        form.reset();
      });
  });
};

const handleError = (err: Error, parent?: HTMLElement) => {
  console.error(err);
  let error = document.createElement("div");
  error.className = "error";
  error.innerText = `Failed to login (${err.name}): ${err.message}`;
  if (!parent) {
    document.body.appendChild(error);
  } else {
    parent.appendChild(error);
  }
};

const oidcToken = (code: string) => {
  let pkce = load_pkce();
  if (pkce == null) {
    return Promise.reject(new Error("failed to load pkce state"));
  }
  let u = new URL("/oauth2/token", OIDC_URL);
  let req = {
    code: code,
    grant_type: "authorization_code",
    client_id: OIDC_CLIENT_ID,
    code_verifier: pkce.verifier,
  };
  return (
    fetch(u.toString(), {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams(req),
    })
      // parse the token and handle errors
      .then(async (res: Response) => {
        if (!res.ok) {
          try {
            const message = await res.json();
            return Promise.reject(new Error(message.error_description));
          } catch (e) {
            return Promise.reject(new Error(res.statusText));
          }
        }
        return res.json();
      })
      // Save token to localStorage and return it
      .then((token: OAuth2Token) => {
        let raw = JSON.stringify(token);
        localStorage.setItem("oidc_token", raw);
        delete_pkce();
        return token;
      })
  );
};

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

main();
