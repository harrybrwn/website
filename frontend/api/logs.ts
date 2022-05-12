import { apiHeaders } from "./common";

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
    `/api/logs?limit=${opts.limit}&offset=${opts.offset}&rev=${opts.reverse}`,
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
