import { loadToken } from "./auth";

export interface PageHits {
  count: number;
}

export const hits = (page: string): Promise<PageHits> => {
  let headers: HeadersInit = {
    Accept: "application/json",
  };
  let token = loadToken();
  if (token != null) {
    headers["Authorization"] = `${token.type} ${token.token}`;
  }
  return fetch(`${window.location.origin}/api/hits?u=${page}`, {
    method: "GET",
    headers: headers,
  })
    .then((res) => {
      if (!res.ok) {
        throw new Error("could not get page hits");
      }
      return res.json();
    })
    .then((blob) => blob as PageHits);
};
