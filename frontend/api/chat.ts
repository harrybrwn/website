import { PageParams } from "./index";

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

export interface MessageResponse {
  messages: Message[];
}

export const messages = async (
  room: number,
  page?: PageParams
): Promise<MessageResponse> => {
  let p: PageParams = page || { limit: 10, offset: 0 };
  let query = `limit=${p.limit}`;
  if (p.offset) query += `&offset=${p.offset}`;
  else if (p.prev) query += `&prev=${p.prev}`;
  return fetch(`/api/chat/${room}/messages?${query}`).then((resp) =>
    resp.json()
  );
};

export const getRoom = async (roomID: number): Promise<Room> => {
  return fetch(`/api/chat/${roomID}`).then((resp) => resp.json());
};
