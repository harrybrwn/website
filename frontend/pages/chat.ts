import "~/frontend/styles/chat.css";

const main = () => {
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

  let chatBody = new ChatBody(body);
  let chatBar = new ChatBar(msgBar, submitMsg);

  chatBar.setMessageHandler((msg: Message) => {
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

const enum Direction {
  Sent,
  Received,
}

interface Message {
  body: string;
  dir: Direction;
}

class ChatBody {
  private container: HTMLElement;

  constructor(container: HTMLElement) {
    this.container = container;
  }

  append(msg: Message) {
    let message = createElement("div", "message");
    let text = createElement("div", "message-text");
    let time = createElement("div", "message-time");

    time.innerText = new Date().toString();
    text.innerText = msg.body;
    switch (msg.dir) {
      case Direction.Sent:
        message.classList.add("sent");
        break;
      case Direction.Received:
        message.classList.add("recv");
        break;
    }
    message.appendChild(text);
    this.container.appendChild(message);
    message.scrollIntoView({
      behavior: "auto",
    });
  }
}

class ChatBar {
  bar: HTMLInputElement;
  button: HTMLButtonElement;
  private handler: ((msg: Message) => void) | null;

  constructor(bar: HTMLInputElement, submit: HTMLButtonElement) {
    this.handler = null;
    this.bar = bar;
    this.button = submit;
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
      this.handler({ body: this.bar.value, dir: Direction.Sent });
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
    if (e.tagName == "INPUT" || e.tagName == "TEXTAREA") return;
    ev.preventDefault();
    switch (ev.key) {
      case "/":
        bar.focus();
        break;
    }
  });
};

main();
