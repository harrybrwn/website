import { loadToken } from "./auth";

export interface PageHits {
  count: number;
}

export const hits = (page: string): Promise<PageHits> => {
  return fetch(`${window.location.origin}/api/hits?u=${page}`, {
    method: "GET",
    headers: apiHeaders(),
  })
    .then((res) => {
      if (!res.ok) {
        throw new Error("could not get page hits");
      }
      return res.json();
    })
    .then((blob) => blob as PageHits);
};

export interface RequestLog {
  id: number;
  method: string;
  status: number;
  ip: string;
  uri: string;
  referer: string;
  user_agent: string;
  latency: number;
  error: string;
  requested_at: string;
}

export const logs = (
  limit: number,
  offset: number,
  reverse: boolean
): Promise<RequestLog[]> => {
  return fetch(
    `${window.location.origin}/api/logs?limit=${limit}&offset=${offset}&rev=${reverse}`,
    {
      method: "GET",
      headers: apiHeaders(),
    }
  ).then((resp) => {
    if (!resp.ok) {
    }
    return resp.json();
  });
};

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
