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

export interface LogOpts {
  limit: number;
  offset?: number;
  reverse?: boolean;
}

export const logs = (opts: LogOpts): Promise<RequestLog[]> => {
  if (opts.limit == undefined) {
    opts.limit = 20;
  }
  if (opts.reverse == undefined) {
    opts.reverse = false;
  }
  if (opts.offset == undefined) {
    opts.offset = 0;
  }

  let u = new URL(window.location.origin);
  u.searchParams.append("limit", opts.limit.toString());
  u.searchParams.append("offset", opts.offset.toString());
  u.searchParams.append("rev", opts.reverse.toString());
  return fetch(
    `${window.location.origin}/api/logs?limit=${opts.limit}&offset=${opts.offset}&rev=${opts.reverse}`,
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

export interface Bookmark {
  url: string;
  name: string;
  description: string;
  tags: string[];
}

export interface Bookmarks {
  links: Bookmark[];
}

/**
 * bookmarks will fetch a list of bookmarks from the api.
 * @returns list of bookmarks
 */
export const bookmarks = async (): Promise<Bookmarks> => {
  return fetch(`${window.location.origin}/api/bookmarks`).then(
    (resp: Response) => {
      if (!resp.ok) {
        throw new Error("could not get bookmarks");
      }
      return resp.json();
    }
  );
};

export interface RuntimeInfo {
  name: string;
  age: number;
  uptime: number;
  goversion: string;
  error: string;
  birthday: string;
  debug: boolean;
  GOOS: string;
  GOARCH: string;
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
