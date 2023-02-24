import { applyTheme } from "~/frontend/components/theme";
import { Bookmark, Bookmarks, bookmarks } from "@hrry.me/api";

import "./styles.css";
import "~/frontend/styles/font.css";
import "~/frontend/components/toggle.css";

const main = () => {
  try {
    applyTheme();
  } catch (e) {}
  let container = document.getElementById("bookmarks") as HTMLElement;
  if (!container) {
    throw new Error("could not get bookmarks container div");
  }
  bookmarks()
    .then((bookmarks: Bookmarks) => {
      for (let bk of bookmarks.links) {
        container.appendChild(bookmarkCard(bk));
      }
    })
    .catch((error) => {
      console.error(error);
      let errMsg = document.createTextNode(
        `Error: unable to get bookmarks: ${error}`
      );
      container.appendChild(errMsg);
    });
};

const bookmarkCard = (bk: Bookmark) => {
  let card = document.createElement("div");
  let a = document.createElement("a");
  let desc = document.createTextNode(bk.description);
  let name = document.createElement("div");
  let tags = document.createElement("div");
  a.href = bk.url;
  a.target = "_blank";
  a.innerText = bk.name;
  a.setAttribute("rel", "noopener, noreferrer");
  name.classList.add("card-name");
  name.appendChild(a);

  for (let t of bk.tags) {
    let tag = document.createElement("span");
    tag.classList.add("bookmark-tag");
    tag.innerText = t;
    tags.appendChild(tag);
  }

  card.classList.add("bookmark");
  card.appendChild(name);
  card.appendChild(desc);
  card.appendChild(tags);
  return card;
};

main();
