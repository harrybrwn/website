export const TOKEN_KEY = "_token";

export interface Token {
  token: string;
  expires: number;
  refresh: string;
  type: string;
}

export interface Claims {
  id: number;
  uuid: string;
  roles: string[];
  aud: string;
  exp: number;
  iat: number;
}

export interface Login {
  username: string;
  email: string;
  password: string;
}

type TokenCallback = (tok: Token) => void;

export const login = async (
  user: Login,
  callback?: TokenCallback
): Promise<Token> => {
  return fetch(`${window.location.origin}/api/token?cookie=true`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(user),
  })
    .then(async (resp: Response) => {
      if (!resp.ok) {
        const message = await resp.json();
        throw new Error(message.message);
      }
      return resp.json();
    })
    .then((blob: any) => {
      let tok: Token = {
        token: blob.token,
        expires: blob.expires,
        refresh: blob.refresh_token,
        type: blob.token_type,
      };
      storeToken(tok);
      if (callback) callback(tok);
      return tok;
    });
};

export const refresh = async (): Promise<Token> => {
  let token = loadToken();
  if (token == null) {
    throw new Error("cannot refresh token when not signed in");
  }
  return fetch(`${window.location.origin}/api/refresh?cookie=true`, {
    method: "POST",
    headers: {
      Accept: "application/json",
      Authorization: `${token.type} ${token.token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ refresh_token: token.refresh }),
  })
    .then((resp) => {
      if (!resp.ok) {
        throw new Error("could not get refresh token");
      }
      return resp.json();
    })
    .then((blob: any) => {
      let tok: Token = {
        token: blob.token,
        expires: blob.expires,
        refresh: blob.refresh_token,
        type: blob.token_type,
      };
      storeToken(tok);
      return tok;
    });
};

export function parseClaims(raw: string): Claims {
  let blob = atob(raw.split(".")[1]);
  return JSON.parse(blob);
}

export function isExpired(tok: Token): boolean {
  return Date.now() >= tok.expires * 1000;
}

export function setCookie(tok: Token) {
  if (document.cookie) {
    document.cookie += "; " + toCookie(tok);
  } else {
    document.cookie = toCookie(tok);
  }
}

export function loadToken(): Token | null {
  let blob = localStorage.getItem(TOKEN_KEY);
  if (blob == null) {
    return null;
  }
  return JSON.parse(blob);
}

export function storeToken(t: Token) {
  localStorage.setItem(TOKEN_KEY, JSON.stringify(t));
}

export function deleteToken() {
  localStorage.removeItem(TOKEN_KEY);
}

function toCookie(tok: Token): string {
  let exp = new Date(tok.expires * 1000);
  let time = exp.getTime();
  let expireTime = time + 1000 * 36000;
  exp.setTime(expireTime);
  let cookie = `${TOKEN_KEY}=${tok.token};expires=${exp.toUTCString()};path=/`;
  return cookie;
}

const apiHeaders = (): HeadersInit => {
  let headers: HeadersInit = {
    Accept: "application/json",
  };
  let token = loadToken();
  if (token != null) {
    headers["Authorization"] = `${token.type} ${token.token}`;
  }
  return headers;
};
