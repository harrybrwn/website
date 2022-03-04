import { Token, storeToken } from "~/frontend/api/auth";

export const mockToken = (): Token => {
  let t = {
    token:
      "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6ImU1Y2NiNmYxLTgxNmYtNGQ2Ny04MjFiLTY0YmU2MDZhZjIyMCIsInJvbGVzIjpbImFkbWluIiwidGFueWEiXSwiaXNzIjoiaGFycnlicnduLmNvbSIsImF1ZCI6WyJ1c2VyIl0sImV4cCI6MTY0NDk5NzM2OSwiaWF0IjoxNjQ0OTk3MzM5fQ.aekMkpbFt96dK-ktqyt4ns8bF5H1NpnxZrnB6EdGFM3c1epIQ97CiC3omeqMzKlktAgD4vrE72LiveiR3nOXDA",
    expires: 1644997369,
    type: "Bearer",
    refresh:
      "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwidXVpZCI6ImU1Y2NiNmYxLTgxNmYtNGQ2Ny04MjFiLTY0YmU2MDZhZjIyMCIsInJvbGVzIjpbImFkbWluIiwidGFueWEiXSwiaXNzIjoiaGFycnlicnduLmNvbSIsImF1ZCI6WyJyZWZyZXNoIl0sImV4cCI6MTY0NTQyOTMzOSwiaWF0IjoxNjQ0OTk3MzM5fQ.RLQfd3k7v5Vy2uvXxdlJ7N5Oq0ruiYzmEXqfDO63qLG1pcDFZWbseg4GAkQTypc-LlE63BrxMnBqRuUxLSSNBg",
  };
  storeToken(t);
  return t;
};
