import "~/frontend/styles/font.css";
import { isEmail } from "~/frontend/util/email";
import { Token, Login, login } from "@hrry.me/api/auth";
import { applyTheme } from "~/frontend/components/theme";

const main = () => {
  applyTheme();
  let form = document.getElementById("login-form") as HTMLFormElement;
  if (form == null) {
    throw new Error("could not find login form");
  }
  form.addEventListener("submit", (ev: SubmitEvent) => {
    ev.preventDefault();
    let d = new FormData(ev.target as HTMLFormElement);
    let ident = d.get("identifier") as string;
    let req: Login = {
      password: d.get("password") as string,
      email: "",
      username: "",
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
