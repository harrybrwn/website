export interface ModalOptions {
  open?: boolean;
  buttonID: string;
  modalID: string;
}

export class Modal {
  open: boolean;
  btn: HTMLElement;
  modal: HTMLElement;
  private opts: ModalOptions;

  private esc: (ev: KeyboardEvent) => void;
  private clk: (ev: MouseEvent) => void;

  constructor(opts: ModalOptions) {
    this.open = opts.open || false;
    this.clk = (_: MouseEvent) => {};
    this.esc = (_: KeyboardEvent) => {};
    this.btn =
      document.getElementById(opts.buttonID) ||
      document.createElement("button");
    this.modal =
      document.getElementById(opts.modalID) || document.createElement("div");
    this.opts = opts;
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
        if (el != null && el.id == this.opts.buttonID) return;
        while (el != null && el != document.body) {
          if (el == this.modal || el.id == this.opts.buttonID) return;
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
}
