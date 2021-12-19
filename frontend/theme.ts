const DEFAULT_TOGGLE_ID = "theme-toggle";

export const getToggle = (id?: string): HTMLElement | null => {
  if (!id) {
    id = DEFAULT_TOGGLE_ID;
  }
  let btn = document.getElementById(id);
  return btn;
};

export enum Theme {
  Dark,
  Light,
}

const loadTheme = (key: string, def?: Theme): Theme => {
  let res = localStorage.getItem(key);
  if (!res) {
    return def || Theme.Dark;
  }
  return parseInt(res);
};

export const applyTheme = () => {
  let toggle = getToggle() as HTMLInputElement;
  let currentTheme = loadTheme("theme");
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)");
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
    if (prefersDark.matches) {
      document.body.classList.toggle("light-theme");
      theme = document.body.classList.contains("light-theme")
        ? Theme.Light
        : Theme.Dark;
    } else {
      document.body.classList.toggle("dark-theme");
      theme = document.body.classList.contains("dark-theme")
        ? Theme.Dark
        : Theme.Light;
    }
    localStorage.setItem("theme", theme.toString());
  });
};
