class Token {
  constructor(blob) {
    this.token = blob.token;
    this.expires = blob.expires;
    this.refresh = blob.refresh_token;
    this.type = blob.token_type;
  }

  toCookie() {
    let exp = new Date(this.expires * 1000);
    let time = exp.getTime();
    let expireTime = time + 1000 * 36000;
    exp.setTime(expireTime);
    let cookie = `_token=${this.token};expires=${exp.toUTCString()};path=/`;
    return cookie;
  }

  setCookie() {
    if (document.cookie) {
      document.cookie += "; " + this.toCookie();
    } else {
      document.cookie = this.toCookie();
    }
  }
}

async function login(username, password) {
  return fetch(`${window.location.origin}/api/token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      username: username,
      password: password,
    }),
  })
    .then((resp) => resp.json())
    .then((blob) => {
      let token = new Token(blob);
      storeToken(token);
      token.setCookie();
      return token;
    });
}

function handleLogin(formID, callback) {
  let form = document.getElementById(formID);
  form.addEventListener("submit", function (event) {
    event.preventDefault();
    let data = {};
    for (const [name, value] of new FormData(form)) {
      data[name] = value;
    }
    console.log(form.action, form.method);
    console.log(data);
    fetch(form.action, {
      method: form.method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    })
      .then((resp) => resp.json())
      .then((blob) => {
        let token = new Token(blob);
        storeToken(token);
        callback(token);
      });
  });
}

function storeToken(token) {
  localStorage.setItem("_token", JSON.stringify(token));
}

function loadToken() {
  return new Token(JSON.parse(localStorage.getItem("_token")));
}

function main() {
  let loginbtn = document.getElementById("login-btn");
  let loginPanel = document.getElementById("login-panel");
  handleLogin("login-form", (token) => {
    console.log(token);
    // token.setCookie();
  });

  let open = false;
  loginbtn.addEventListener("click", (event) => {
    console.log(event);
    if (open) {
      loginPanel.style.display = "none";
      open = false;
    } else {
      loginPanel.style.display = "block";
      open = true;
    }
  });
}
