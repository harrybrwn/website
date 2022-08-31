import "~/frontend/styles/font.css";
import { isEmail } from "~/frontend/util/email";
import { applyTheme } from "~/frontend/components/theme";

interface RedirectTarget {
  redirect_to: string;
}

interface LoginRequest {
  password: string;
  email: string;
  username: string;
  login_challenge?: string | null;
}

const main = () => {
  try {
    applyTheme();
  } catch (err) {}
  let form = document.getElementById("login-form") as HTMLFormElement;
  if (form == null) {
    throw new Error("could not find login form");
  }
  const oidcbtn = document.getElementById("with-oidc");
  if (oidcbtn == null) {
    console.warn('could not find "login with oidc" button');
  } else {
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
    code_flow(code, form);
    return;
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

const login = async (req: LoginRequest): Promise<RedirectTarget> => {
  return fetch("/api/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  }).then(async (res: Response) => {
    if (!res.ok) {
      try {
        const message = await res.json();
        return Promise.reject(new Error(message.message));
      } catch (e) {
        return Promise.reject(new Error(res.statusText));
      }
    }
    return res.json();
  });
};

const consent = async (challenge: string): Promise<RedirectTarget> => {
  return fetch("/api/consent", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ consent_challenge: challenge }),
  }).then((res: Response) => res.json());
};

const code_flow = (code: string, form: HTMLFormElement) => {
  let pkce = load_pkce();
  if (pkce == null) {
    let e = new Error("failed to load pkce state");
    handleError(e, form);
    throw e;
  }
  let u = new URL("/oauth2/token", OIDC_URL);
  let req = {
    code: code,
    grant_type: "authorization_code",
    client_id: OIDC_CLIENT_ID,
    code_verifier: pkce.verifier,
  };
  fetch(u.toString(), {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: new URLSearchParams(req),
  })
    .then(async (res: Response) => {
      if (!res.ok) {
        try {
          const message = await res.json();
          console.error(message);
          return Promise.reject(new Error(message.error_description));
        } catch (e) {
          return Promise.reject(new Error(res.statusText));
        }
      }
      return res.json();
    })
    .then((blob) => {
      let raw = JSON.stringify(blob);
      let node = document.createElement("p");
      node.innerText = raw;
      document.body.appendChild(node);
      localStorage.setItem("oidc_token", raw);
      delete_pkce();
      window.location.pathname = "/";
    })
    .catch(handleError);
};

interface Pkce {
  verifier: string;
  challenge: string;
  state: string;
}

const new_pkce = async (): Promise<Pkce> => {
  const verifierBuf = crypto.getRandomValues(new Uint8Array(96));
  const verifier = b64(verifierBuf); // this will have a length of 128

  const ascii = new Uint8Array(verifier.length);
  for (let i = 0; i < verifier.length; i++) ascii[i] = verifier.charCodeAt(i);

  const hash = await crypto.subtle.digest("SHA-256", ascii);
  return {
    verifier: verifier,
    challenge: b64(new Uint8Array(hash)),
    state: b64(crypto.getRandomValues(new Uint8Array(16))),
  };
};

const save_pkce = (pkce: Pkce) => {
  localStorage.setItem("oidc_verifier", pkce.verifier);
  localStorage.setItem("oidc_challenge", pkce.challenge);
  localStorage.setItem("oidc_state", pkce.state);
};

const delete_pkce = () => {
  localStorage.removeItem("oidc_verifier");
  localStorage.removeItem("oidc_challenge");
  localStorage.removeItem("oidc_state");
};

const load_pkce = (): Pkce | null => {
  let v = localStorage.getItem("oidc_verifier");
  let c = localStorage.getItem("oidc_challenge");
  let s = localStorage.getItem("oidc_state");
  if (v == null || c == null || s == null) {
    console.warn("failed to load oidc state:", v, c, s);
    return null;
  }
  return {
    verifier: v,
    challenge: c,
    state: s,
  };
};

const b64 = (buf: Uint8Array) => {
  // https://tools.ietf.org/html/rfc4648#section-5
  return btoa(String.fromCharCode.apply(null, Array.from(buf)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
};

main();
