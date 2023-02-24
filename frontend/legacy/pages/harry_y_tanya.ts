import { applyTheme } from "~/frontend/components/theme";
import LoginManager from "~/frontend/util/LoginManager";
import "~/frontend/styles/font.css";
import "~/frontend/components/toggle.css";

try {
  applyTheme();
} catch (error) {}

let m = new LoginManager();

document.addEventListener("tokenChange", (ev: TokenChangeEvent) => {
  console.log(ev.detail.action);
});
