export interface Message {
  id: number;
  room: number;
  user_id: number;
  body: string;
  created_at: Date;
}
