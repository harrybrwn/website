export const TOKEN_KEY = "_token";

export interface Token {
  token: string;
  expires: number;
  refresh: string;
  type: string;
}

export interface Login {
  username: string;
  email: string;
  password: string;
}

type TokenCallback = (tok: Token) => void;

export function login(user: Login, callback?: TokenCallback): Promise<Token> {
  return fetch(`${window.location.origin}/api/token?cookie=true`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(user),
  })
    .then((resp: Response) => resp.json())
    .then((blob: any) => {
      storeToken(blob);
      if (callback) callback(blob);
      return blob;
    });
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
