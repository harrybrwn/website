import "./theme.css";

const DEFAULT_TOGGLE_ID = "theme-toggle";

const getToggle = (id?: string): HTMLElement | null => {
  if (!id) {
    id = DEFAULT_TOGGLE_ID;
  }
  let btn = document.getElementById(id);
  return btn;
};

const loadTheme = (key: string): Theme => {
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

export class ThemeManager {
  theme: Theme;
  themeToggle: HTMLInputElement;

  constructor() {
    this.theme = loadTheme("theme");
    this.themeToggle = getToggle() as HTMLInputElement;
    if (!this.themeToggle) {
      console.error("could not find theme toggle");
    }
    switch (this.theme) {
      case Theme.Dark:
        document.body.classList.toggle("dark-theme");
        break;
      case Theme.Light:
        document.body.classList.toggle("light-theme");
        this.themeToggle.checked = true;
        break;
    }
  }

  toggle() {
    if (this.theme == Theme.Dark) {
      document.body.classList.remove("dark-theme");
      document.body.classList.add("light-theme");
      this.theme = Theme.Light;
    } else {
      document.body.classList.remove("light-theme");
      document.body.classList.add("dark-theme");
      this.theme = Theme.Dark;
    }
    // this.themeToggle.checked = !this.themeToggle.checked;
    localStorage.setItem("theme", this.theme.toString());
  }

  onChange(fn: (ev: Event) => void) {
    this.themeToggle.addEventListener("change", fn);
  }
}

export const applyTheme = () => {
  let man = new ThemeManager();
  man.onChange((ev: Event) => {
    man.toggle();
  });
};
