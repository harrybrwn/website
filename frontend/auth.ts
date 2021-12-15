interface Token {
  token: string;
}

export function login(username: string, password: string): Token | null {
  console.log("logging in...");
  return null;
}
