export const websocketURL = (path: string): URL => {
  const protocol = location.protocol == "https:" ? "wss" : "ws";
  if (path[0] != "/") path = `/${path}`;
  return new URL(`${protocol}://${location.host}${path}`);
};
