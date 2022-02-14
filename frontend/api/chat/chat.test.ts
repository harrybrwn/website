describe("utilities", () => {
  test("websocket url generator", () => {
    let url = websocketURL("/path/to/websocket");
    expect(url.protocol).toBe("ws");
  });
});
