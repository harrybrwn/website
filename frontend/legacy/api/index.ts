import { Role } from "./auth";
import { apiHeaders } from "./common";

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

export interface InviteRequest {
  timeout?: number;
  ttl?: number;
  email?: string;
  receiver_name?: string;
  // List of string roles because the server will parse them into the role enum.
  roles?: string[];
}

export interface InviteURL {
  path: string;
  created_by: string;
  expires_at: string;
  email: string;
  receiver_name: string;
  roles: Role[];
  ttl: number;
}

export interface InviteList {
  invites: InviteURL[];
}

export const invite = async (req?: InviteRequest): Promise<InviteURL> => {
  let body: string;
  if (req) {
    body = JSON.stringify(req);
  } else {
    body = "";
  }
  return fetch("/api/invite/create", {
    method: "POST",
    headers: apiHeaders(),
    body: body,
  }).then(async (res) => {
    if (!res.ok) {
      let msg = await res.json();
      throw new Error(msg.message);
    }
    return res.json();
  });
};

export const invites = async (): Promise<InviteList> => {
  return fetch("/api/invites", {
    method: "GET",
    headers: apiHeaders(),
  }).then((res) => {
    if (!res.ok) {
      return Promise.reject(res.statusText);
    }
    return res.json();
  });
};

export interface PageParams {
  limit: number;
  offset?: number;
  prev?: number;
}

export interface RuntimeInfo {
  name: string;
  age: number;
  uptime: number;
  goversion: string;
  error: string;
  birthday: string;
  debug: boolean;
  build: { [key: string]: string };
}

/**
 * Retrieve the api server's runtime info.
 * @returns the server's runtime info
 */
export const runtimeInfo = async (): Promise<RuntimeInfo> => {
  return fetch(`${window.location.origin}/api/runtime`, {
    method: "GET",
    headers: apiHeaders(),
  }).then((resp) => {
    if (!resp.ok) {
      throw new Error("could not get runtime info");
    }
    return resp.json();
  });
};

export type { Bookmarks, Bookmark } from "./bookmarks";
export type { RequestLog, LogOpts } from "./logs";
export { bookmarks } from "./bookmarks";
export { login, refresh } from "./auth";
export { logs } from "./logs";
