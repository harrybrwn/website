import "./toggle.css";

interface SwitchProps {
  tag?: string;
  id?: string;
}

export const Switch = (props: SwitchProps): HTMLElement => {
  let outer = document.createElement(props.tag || "span");
  let check = document.createElement("input");
  let label = document.createElement("label");
  check.setAttribute("type", "checkbox");
  if (props.id) {
    check.setAttribute("id", props.id);
    label.setAttribute("for", props.id);
  }
  outer.appendChild(check);
  outer.appendChild(label);
  return outer;
};

export const GetSwitch = (
  id: string,
  props?: SwitchProps
): HTMLElement | null => {
  // let switch = document.getElementById(id);
  // if (!switch) return switch;
  let s = document.getElementById(id);
  if (!props) {
    return s;
  }
  return s;
};
