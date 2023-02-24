import { loadToken } from "./auth";

export const apiHeaders = (): HeadersInit => {
  let headers: HeadersInit = {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
  let token = loadToken();
  if (token != null) {
    headers["Authorization"] = `${token.type} ${token.token}`;
  }
  return headers;
};
