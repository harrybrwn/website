import { websocketURL } from "~/frontend/util/websocket";

describe("utilities", () => {
  test("websocket url generator", () => {
    let url = websocketURL("/path/to/websocket");
    expect(url.protocol).toBe("ws:");
    expect(url.host).toBe("localhost");
  });
});
