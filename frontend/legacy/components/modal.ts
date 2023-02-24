export interface ModalOptions {
  open?: boolean;
  element: HTMLElement | null;
  button?: HTMLElement | null;
}

export class Modal {
  open: boolean;
  button: HTMLElement;
  modal: HTMLElement;

  private esc: (ev: KeyboardEvent) => void;
  private clk: (ev: MouseEvent) => void;

  constructor(opts: ModalOptions) {
    if (opts.element == null) {
      throw new Error("modal element is null");
    }
    this.open = opts.open || false;
    this.clk = (_: MouseEvent) => {};
    this.esc = (_: KeyboardEvent) => {};
    this.modal = opts.element;
    this.button = opts.button || document.createElement("button");
  }

  private _toggle() {
    this.modal.style.display = this.open ? "none" : "block";
    this.open = !this.open;
  }

  private cleanup() {
    window.removeEventListener("click", this.clk);
    window.removeEventListener("keydown", this.esc);
  }

  toggle() {
    if (!this.open) {
      this.clk = (ev: MouseEvent) => {
        let el = ev.target as HTMLElement | null;
        if (el != null && el.id == this.button.id) {
          return;
        }
        while (el != null && el != document.body) {
          if (el == this.modal || el.id == this.button.id) {
            return;
          }
          el = el.parentElement;
        }
        this._toggle();
        this.cleanup();
      };
      this.esc = (ev: KeyboardEvent) => {
        if (ev.key != "Escape") {
          return;
        }
        this._toggle();
        this.cleanup();
      };
    }
    this._toggle();
    if (this.open) {
      window.addEventListener("click", this.clk);
      window.addEventListener("keydown", this.esc);
    } else {
      this.cleanup();
    }
  }

  onClick(cb: (ev: MouseEvent) => void) {
    this.button.addEventListener("click", cb);
  }

  toggleOnClick() {
    this.button.addEventListener("click", (ev: MouseEvent) => {
      this.toggle();
    });
  }
}
