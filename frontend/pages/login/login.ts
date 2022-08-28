import "~/frontend/styles/font.css";
import { isEmail } from "~/frontend/util/email";
import { Token, Login, login } from "@hrry.me/api/auth";
import { applyTheme } from "~/frontend/components/theme";

console.log("hello");
const main = () => {
  try {
    applyTheme();
  } catch (err) {}
  let form = document.getElementById("login-form") as HTMLFormElement;
  if (form == null) {
    throw new Error("could not find login form");
  }
  const params = new URLSearchParams(window.location.search);

  form.addEventListener("submit", (ev: SubmitEvent) => {
    ev.preventDefault();
    let d = new FormData(ev.target as HTMLFormElement);
    let ident = d.get("identifier") as string;
    let req = {
      password: d.get("password") as string,
      email: "",
      username: "",
      login_challenge: params.get("login_challenge"),
      consent_challenge: params.get("consent_challenge"),
    };
    if (isEmail(ident)) {
      req.email = ident;
    } else {
      req.username = ident;
    }
    login(req)
      .then((token: Token) => {
        window.location.pathname = "/";
      })
      .catch((err: Error) => {
        console.error(err);
        let error = document.createElement("div");
        error.className = "error";
        error.innerText = `Failed to login: ${err.message}`;
        form.appendChild(error);
        form.reset();
      });
  });
};

main();
