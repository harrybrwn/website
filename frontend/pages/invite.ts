import "~/frontend/styles/font.css";
import { isEmail } from "~/frontend/util/email";
import { millisecondsToStr } from "~/frontend/util/time";

const main = () => {
  let metaBlob = document.getElementById("invite-meta");
  if (metaBlob == null) {
    throw new Error("could not find invite meta data");
  }
  let meta = JSON.parse(metaBlob.innerText);
  let expires = new Date(meta.expires_at).getTime();
  let timerText = document.getElementById("self-destruct");
  if (timerText == null) {
    throw new Error("no timer element");
  }

  let form = document.getElementById("signup-form");
  if (form == null) throw new Error("could not find login form");
  handleFormActions(form as HTMLFormElement);

  let timer: NodeJS.Timer;
  const update = () => {
    timerText = timerText as HTMLElement;
    let exp = expires - Date.now();
    if (exp <= 0) {
      clearInterval(timer);
      selfDestruct();
      return;
    }
    timerText.innerText = millisecondsToStr(exp);
  };
  update();
  timer = setInterval(update, 1000);
};

const selfDestruct = () => {
  let msg = document.getElementById("self-destruct-msg");
  if (msg != null) {
    msg.innerText = "This page has self destructed.";
  }
};

const handleFormActions = (form: HTMLFormElement) => {
  let errorContainer = document.createElement("div");
  form.parentElement?.appendChild(errorContainer);

  form.addEventListener("submit", (ev: SubmitEvent) => {
    ev.preventDefault();
    let target = ev.target as HTMLFormElement;
    let data = new FormData(form as HTMLFormElement);
    let body = {
      username: data.get("username") as string,
      email: data.get("email") as string,
      password: data.get("password") as string,
    };
    if (!isEmail(body.email)) {
      errorContainer.innerText = "Error: invalid email";
      return;
    }
    if (body.username.length == 0) {
      errorContainer.innerText = "Error: no username";
      return;
    }
    if (body.password.length < 5) {
      errorContainer.innerText =
        "Error: password must be more that 5 characters";
      return;
    }
    fetch(target.action, {
      method: target.method,
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    }).then(async (res) => {
      if (!res.ok) {
        let msg = await res.json();
        errorContainer.innerText = `Error: ${msg}`;
        return;
      }
      window.location.pathname = "/";
    });
  });
};

main();
