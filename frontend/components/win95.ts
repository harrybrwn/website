interface Win95Props {
  head?: HTMLElement[];
  body?: HTMLElement[];
  id?: string;
}

function Win95Window(props?: Win95Props) {
  if (props && props.id) {
    let win = document.getElementById(props.id);
    if (win == null) {
      throw new Error(`could not find ${props.id}`);
    }
    return win;
  }

  let head = document.createElement("div");
  head.classList.add("win95-window-head");
  let body = document.createElement("div");
  body.classList.add("win95-window-body");

  if (props) {
    if (props.head) {
      for (let i in props.head) {
        head.appendChild(props.head[i]);
      }
    }
    if (props.body) {
      for (let i in props.body) {
        body.appendChild(props.body[i]);
      }
    }
  }

  let win = document.createElement("div");
  win.classList.add("win95-window");
  win.appendChild(head);
  win.appendChild(body);
  return win;
}
