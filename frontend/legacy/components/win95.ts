import "./win95.css";

interface Win95Props {
  head?: HTMLElement[];
  body?: HTMLElement[];
  id?: string;
}

class Win95Window extends HTMLElement {
  head: Element;
  body: Element;

  constructor() {
    super();
    for (let i = 0; i < this.children.length; i++) {
      console.log(this.children[i]);
    }
    this.head = this.children[0] || document.createElement("div");
    this.body = this.children[1] || document.createElement("div");
    this.classList.add("win95-window");
    this.head.classList.add("win95-window-head");
    this.body.classList.add("win95-window-body");
  }
}

customElements.define("win95-window", Win95Window, { extends: "div" });
// customElements.define("win95-window", Win95Window);
