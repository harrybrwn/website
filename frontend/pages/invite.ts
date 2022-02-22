import "~/frontend/styles/font.css";
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

main();
