// NOTE: This will not clear cookies with HttpOnly set and will
// not clear cookies with the Path value set.
export function clearCookie(name: string) {
  // This is really gross, but it works
  // https://stackoverflow.com/questions/179355/clearing-all-cookies-with-javascript
  document.cookie = name + "=;expires=Thu, 01 Jan 1970 00:00:00 GMT";
}
