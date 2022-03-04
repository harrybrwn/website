export const TOKEN_KEY = "_token";
export const REFRESH_TOKEN_KEY = "_refresh";

export interface Token {
  token: string;
  expires: number;
  refresh: string;
  type: string;
}

export type Role = string;

export interface Claims {
  id: number;
  uuid: string;
  roles: Role[];
  aud: string[];
  iss: string;
  exp: number;
  iat: number;
}

export interface Login {
  username: string;
  email: string;
  password: string;
}

export const login = async (user: Login): Promise<Token> => {
  return fetch("/api/token?cookie=true", {
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
      return tok;
    });
};

export const refresh = async (refresh: string): Promise<Token> => {
  return fetch("/api/refresh?cookie=true", {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ refresh_token: refresh }),
  })
    .then((resp) => {
      if (!resp.ok) {
        // const msg = await resp.json();
        // throw new Error(`could not get new access token: ${msg.message}`);
        throw new Error(
          `could not get new access token: status ${resp.status}`
        );
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

export const loadRefreshToken = (): string | null => {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
};

/**
 * This will store the refresh token separately if it needs to otherwise will
 * not touch the refresh token.
 * @param t is the jwt token being stored
 */
export function storeToken(t: Token) {
  let refresh = localStorage.getItem(REFRESH_TOKEN_KEY);
  if (
    (refresh == null || refresh != t.refresh) &&
    t.refresh != null &&
    t.refresh.length > 0
  ) {
    localStorage.setItem(REFRESH_TOKEN_KEY, t.refresh);
  }
  // don't need to store the refresh token twice
  localStorage.setItem(
    TOKEN_KEY,
    JSON.stringify({
      token: t.token,
      expires: t.expires,
      type: t.type,
    })
  );
}

export function deleteToken() {
  localStorage.removeItem(TOKEN_KEY);
}

export const clearRefreshToken = () => {
  localStorage.removeItem(REFRESH_TOKEN_KEY);
};

export const refreshExpiration = (): Date | null => {
  let t = loadRefreshToken();
  if (t == null) {
    return null;
  }
  let claims = parseClaims(t);
  return new Date(claims.exp * 1000);
};

function toCookie(tok: Token): string {
  let exp = new Date(tok.expires * 1000);
  let time = exp.getTime();
  let expireTime = time + 1000 * 36000;
  exp.setTime(expireTime);
  let cookie = `${TOKEN_KEY}=${tok.token};expires=${exp.toUTCString()};path=/`;
  return cookie;
}

export const authHeader = (): string => {
  let token = loadToken();
  if (token == null) {
    return "";
  }
  return `${token.type} ${token.token}`;
};

const apiHeaders = (): HeadersInit => {
  return {
    Accept: "application/json",
    Authorization: authHeader(),
  };
};
