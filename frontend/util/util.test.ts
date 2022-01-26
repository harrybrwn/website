import { clearCookie } from "./cookies";

test("this is a test", () => {
  let name = "key";
  let d = new Date(Date.now() + 100000000);
  document.cookie = `${name}=value;expires=${d};path=/`;
  expect(document.cookie).toBe(`${name}=value`);
  clearCookie(name);
  expect(document.cookie).toEqual("");
});
