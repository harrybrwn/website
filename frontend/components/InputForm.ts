type ValidationFunction = (key: string, val: string | Object) => Error;

interface InputFormOptions {
  action: string;
  method: string;
  enctype: string;
  validate?: ValidationFunction;
}

type FormDataObject = { [key: string]: string | Object };

export default class InputForm extends HTMLFormElement {
  validate?: ValidationFunction;

  constructor(opts?: InputFormOptions) {
    super();
    if (opts) {
      this.action = opts.action;
      this.method = opts.method;
      this.enctype = opts.enctype;
      this.encoding = opts.enctype;
      if (opts.validate) this.validate = opts.validate;
    }
    this.addEventListener("submit", this.apiCall);
  }

  async apiCall(ev: SubmitEvent) {
    ev.preventDefault();
    let opts = {
      method: this.method,
      headers: {
        "Content-Type": this.enctype,
      },
      body: this.encode(this.enctype, new FormData(this)),
    };
    let res = await fetch(this.action, opts);
    return res;
  }

  formDataBody(data: FormData): FormDataObject {
    let obj: FormDataObject = {};
    data.forEach((entry: FormDataEntryValue, key: string, parent: FormData) => {
      obj[key] = entry.valueOf();
    });
    return obj;
  }

  encode(type: string, data: FormData): string | null {
    switch (type) {
      case "application/json":
        return JSON.stringify(this.formDataBody(data));
      default:
        return null;
    }
  }
}

customElements.define("input-form", InputForm, { extends: "form" });
