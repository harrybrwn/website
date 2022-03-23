import "~/frontend/styles/chat.css";
import "~/frontend/components/toggle.css";
import { websocketURL } from "~/frontend/util/websocket";
import { SECOND } from "~/frontend/constants";
import { Message, Room } from "~/frontend/api/chat";
import { ThemeManager } from "~/frontend/components/theme";
import {
  TOKEN_KEY,
  loadToken,
  parseClaims,
  deleteToken,
} from "~/frontend/api/auth";
import { clearCookie } from "~/frontend/util/cookies";
import LoginManager from "~/frontend/util/LoginManager";

const main = () => {
  let themeManager = new ThemeManager();
  let loginManager = new LoginManager({
    interval: 5 * 60 * SECOND,
    clearToken: () => {
      deleteToken();
      clearCookie(TOKEN_KEY);
    },
  });
  let msgBar = document.getElementById(
    "new-msg-textbar"
  ) as HTMLInputElement | null;
  let submitMsg = document.getElementById(
    "submit-new-msg"
  ) as HTMLButtonElement | null;
  if (msgBar == null) throw new Error("no message bar");
  if (submitMsg == null) throw new Error("no message submit");

  let body = document.getElementById("conversation-messages");
  if (body == null) throw new Error("no message body");

  let userID: number;
  let t = loadToken();
  if (t != null) {
    let claims = parseClaims(t.token);
    userID = claims.id;
  } else {
    userID = Math.round(Math.random() * 1000);
  }
  let roomID = parseInt(location.pathname.split("/")[2]);

  let chatBody = new ChatBody(userID, body);
  let chatBar = new ChatBar({
    userID,
    roomID,
    bar: msgBar,
    submit: submitMsg,
  });

  let conn = new WebSocket(
    websocketURL(`/api/chat/${roomID}/connect?user=${userID}`)
  );
  let open = true;
  conn.onmessage = (ev: MessageEvent) => {
    console.log("received message:", ev.data, ev.origin);
    let msg: Message = JSON.parse(ev.data);
    chatBody.append(msg);
  };
  conn.onclose = (ev: CloseEvent) => {
    console.warn("socket has closed:", ev.reason);
    open = false;
  };
  conn.onerror = (ev: Event) => {
    console.error("websocket error:", ev);
  };
  chatBar.setMessageHandler((msg: Message) => {
    if (!open) {
      // TODO put an error in front of the user
      console.error("websocket is closed. cannot send message");
      return;
    }
    console.log("sending message:", msg);
    let message = JSON.stringify(msg);
    conn.send(message);
    chatBody.append(msg);
  });

  handleKeyPresses(chatBar);

  // Scroll to the bottom of the conversation on window load.
  window.addEventListener("load", (ev: Event) => {
    let m = document.getElementById("conversation-messages");
    if (m == null || m.lastElementChild == null) return;
    m.lastElementChild.scrollIntoView({ behavior: "auto" });
  });
};

class ChatBody {
  private container: HTMLElement;
  private userID: number;
  private room: Room | null;

  constructor(userID: number, container: HTMLElement) {
    this.userID = userID;
    this.container = container;
    this.room = null;
  }

  append(msg: Message) {
    let message = createElement("div", "message");
    let text = createElement("div", "message-text");
    let time = createElement("div", "message-time");
    let username = createElement("span", "msg-username");

    if (this.room != null) {
      let member = this.room.members.get(msg.user_id);
      username.innerText = member ? member.username : `${msg.user_id}`;
    } else {
      username.innerText = `${msg.user_id}`;
    }

    time.innerText = new Date().toString();
    text.innerText = msg.body;
    if (msg.user_id == this.userID) {
      message.classList.add("sent");
    } else {
      message.classList.add("recv");
    }
    message.appendChild(text);
    message.appendChild(username);
    this.container.appendChild(message);
    // Scroll down to show the new message
    message.scrollIntoView({
      behavior: "auto",
    });
  }
}

class ChatBar {
  bar: HTMLInputElement;
  button: HTMLButtonElement;
  private handler: ((msg: Message) => void) | null;
  private userID: number;
  private roomID: number;

  constructor(opts: {
    userID: number;
    roomID: number;
    bar: HTMLInputElement;
    submit: HTMLButtonElement;
  }) {
    this.userID = opts.userID;
    this.roomID = opts.roomID;
    this.handler = null;
    this.bar = opts.bar;
    this.button = opts.submit;
    // Submit new message via send button
    this.button.addEventListener("click", (ev: MouseEvent) => this.message(ev));
    // Submit new message via enter key.
    this.bar.addEventListener("keypress", (ev: KeyboardEvent) => {
      // Submit when pressing Enter
      if (ev.key == "Enter" && !ev.shiftKey) {
        ev.preventDefault();
        this.message(ev);
        return;
      }
    });
  }

  private message(ev: Event) {
    if (this.bar.value.length == 0) {
      return;
    }
    if (this.handler != null) {
      this.handler({
        id: 0, // we do not know what the message id is yet
        room: this.roomID,
        body: this.bar.value,
        user_id: this.userID,
        created_at: new Date(),
      });
    }
    this.bar.value = "";
  }

  setMessageHandler(h: (msg: Message) => void) {
    this.handler = h;
  }

  focus() {
    this.bar.focus();
  }
}

const createElement = (
  type: keyof HTMLElementTagNameMap,
  className: string
) => {
  let el = document.createElement(type);
  el.classList.add(className);
  return el;
};

const handleKeyPresses = (bar: ChatBar) => {
  document.addEventListener("keydown", (ev: KeyboardEvent) => {
    const e = ev.target as HTMLElement;
    if (e.tagName == "INPUT" || e.tagName == "TEXTAREA") {
      return;
    }
    switch (ev.key) {
      case "/":
        ev.preventDefault();
        bar.focus();
        break;
    }
  });
};

main();
