import { PageParams } from "./index";
import { authHeader } from "./auth";

export interface Message {
  id: number;
  room: number;
  user_id: number;
  body: string;
  created_at: Date;
}

export interface Room {
  id: number;
  owner_id: number;
  name: string;
  public: boolean;
  created_at: Date;
  // members: ChatMember[];
  members: Map<number, ChatMember>;
}

export interface ChatMember {
  room: number;
  user_id: number;
  last_seen: number;
  username: string;
}

export interface MessagesResponse {
  messages: Array<Message>;
}

export const messages = async (
  room: number,
  page?: PageParams
): Promise<MessagesResponse> => {
  let p: PageParams = page || { limit: 10, offset: 0 };
  let query = `limit=${p.limit}`;
  if (p.offset) query += `&offset=${p.offset}`;
  else if (p.prev) query += `&prev=${p.prev}`;

  return fetch(`/api/chat/${room}/messages?${query}`, {
    headers: {
      Accept: "application/json",
      Authorization: authHeader(),
    },
  }).then(async (resp) => {
    if (!resp.ok) {
      const msg = await resp.json();
      throw new Error(msg);
    }
    let msgs: MessagesResponse = await resp.json();
    for (let i = 0; i < msgs.messages.length; i++) {
      msgs.messages[i].created_at = new Date(msgs.messages[i].created_at);
    }
    return msgs;
  });
};

const msgSort = (a: Message, b: Message): number => {
  return a.created_at.getTime() - b.created_at.getTime();
};

export const getRoom = async (roomID: number): Promise<Room> => {
  return fetch(`/api/chat/${roomID}`).then(async (resp) => {
    if (!resp.ok) {
      const msg = await resp.json();
      throw new Error(msg);
    }
    return resp.json();
  });
};
