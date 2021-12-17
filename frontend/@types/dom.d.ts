type TokenChangeEvent = CustomEvent<{
  signedIn: boolean;
  token: Token | null;
  readonly action: "login" | "logout";
}>;

// type LoggedInEvent = CustomEvent<{
//   signedIn: boolean;
//   token: Token | null;
// }>;

// TODO support login and logout handlers
interface TokenChangeEventHandlersEventMap {
  tokenChange: TokenChangeEvent;

  loggedIn: TokenChangeEvent;
  loggedOut: TokenChangeEvent;
}

interface Document {
  addEventListener<K extends keyof TokenChangeEventHandlersEventMap>(
    type: K,
    listener: (this: Document, ev: TokenChangeEventHandlersEventMap[K]) => any,
    options?: boolean | AddEventListenerOptions
  ): void;

  addEventListener<K extends keyof TokenChangeEventHandlersEventMap>(
    type: K,
    listener: (this: Document, ev: TokenChangeEventHandlersEventMap[K]) => any
  ): void;
}

declare module "*.gif" {
  const value: any;
  export default value;
}

declare module "*.jpg" {
  const value: any;
  export default value;
}

declare module "*.svg" {
  const value: any;
  export default value;
}

declare module "*.html" {
  const value: any;
  export default value;
}
