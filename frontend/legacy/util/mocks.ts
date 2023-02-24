import { Token, Role, storeToken, Claims } from "@hrry.me/api/auth";

export const mockToken = (): Token => {
  let t = newToken({
    id: 1,
    uuid: "e5ccb6f1-816f-4d67-821b-64be606af220",
    roles: [Role.Admin],
    iss: "harrybrwn.com",
    aud: ["user"],
    exp: 1644997369,
    iat: 1644997339,
  });
  storeToken(t);
  return t;
};

const b64encode = (s: string): string => {
  let b = btoa(s);
  return b.replace("==", "").replace("=", "");
};

export const newToken = (claims: Claims, refExp?: number): Token => {
  let exp = Math.round(claims.exp);
  let cl = {
    ...claims,
  };
  const base = "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9";
  const sig =
    "RLQfd3k7v5Vy2uvXxdlJ7N5Oq0ruiYzmEXqfDO63qLG1pcDFZWbseg4GAkQTypc-LlE63BrxMnBqRuUxLSSNBg";
  let encClaims = b64encode(JSON.stringify(cl));

  if (refExp != undefined) cl.exp = Math.round(refExp);
  else cl.exp = cl.exp + 24 * 60 * 60;
  cl.aud = ["refresh"];
  let encRefClaims = b64encode(JSON.stringify(cl));

  return {
    type: "Bearer",
    expires: exp,
    token: base + "." + encClaims + "." + sig,
    refresh: base + "." + encRefClaims + "." + sig,
  };
};
