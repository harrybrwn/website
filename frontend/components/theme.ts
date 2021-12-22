import "./theme.css";

const DEFAULT_TOGGLE_ID = "theme-toggle";

const getToggle = (id?: string): HTMLElement | null => {
  if (!id) {
    id = DEFAULT_TOGGLE_ID;
  }
  let btn = document.getElementById(id);
  return btn;
};

const loadTheme = (key: string): Theme | null => {
  let res = localStorage.getItem(key);
  if (res == null) {
    const prefersDark = window.matchMedia("(prefers-color-scheme: dark)");
    return prefersDark.matches ? Theme.Dark : Theme.Light;
  }
  return parseInt(res);
};

export enum Theme {
  Dark,
  Light,
}

export const applyTheme = () => {
  let toggle = getToggle() as HTMLInputElement | null;
  if (toggle == null) {
    toggle = document.createElement("input");
  }
  let currentTheme = loadTheme("theme");
  switch (currentTheme) {
    case Theme.Dark:
      document.body.classList.toggle("dark-theme");
      break;
    case Theme.Light:
      document.body.classList.toggle("light-theme");
      toggle.checked = true;
      break;
  }
  toggle.addEventListener("change", (ev: Event) => {
    let theme: Theme;
    if (currentTheme == Theme.Dark) {
      theme = Theme.Light;
      document.body.classList.remove("dark-theme");
      document.body.classList.add("light-theme");
    } else {
      theme = Theme.Dark;
      document.body.classList.remove("light-theme");
      document.body.classList.add("dark-theme");
    }
    currentTheme = theme;
    localStorage.setItem("theme", theme.toString());
  });
};
