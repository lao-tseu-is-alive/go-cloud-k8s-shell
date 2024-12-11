import "./skeleton.css";
import "@xterm/xterm/css/xterm.css";
import { APP, BUILD_DATE, REPOSITORY, VERSION } from "./version.ts";
import { setupTerminal } from "./terminal.ts";

const html = `
<div class="container">
  <section class="header">
     <h6 id="appInfo">app</h6>
  </section>  
  <form method="post" action="/login" id="loginForm">
        <div class="row">
            <div class="six columns">
                <label for="login">Login:</label><br/>
                <input id="login" type="text" name="login" value="" class="u-full-width">
            </div>
            <div class="six columns">
                <label for="password">Password:</label><br/>
                <input id="password" type="password"
                       placeholder="Enter your password here" class="u-full-width">
                <input type="hidden" id="hashed" name="hashed" >
            </div>
        </div>
        <input type="submit" class="u-pull-right">
    </form>
    <div class="row" id="divMsg">
        <div class="twelve columns">
            <div id="msg"></div>
        </div>
    </div>
  <div class="row">
  <div class="row">
    <div class="twelve columns">
      <div id="terminal"></div>
    </div>    
  </div>  
</div>

`;
document.querySelector<HTMLDivElement>("#app")!.innerHTML = html;
// if token is null, display login form
const appInfo = document.getElementById("appInfo")!;
const loginForm: HTMLFormElement = document.getElementById(
  "loginForm",
)! as HTMLFormElement;
const msg = document.getElementById("msg")!;
const divMsg: HTMLDivElement =
  document.querySelector<HTMLDivElement>("#divMsg")!;
let token = null;
const serverHost =
  window.location.port === "5173" ? "localhost:9999" : window.location.host;
appInfo.innerHTML = `<a href="${REPOSITORY}">${APP}</a> v${VERSION} - ${BUILD_DATE}`;

async function hashPasswordSha256(password:string) {
  const encoder = new TextEncoder();
  const data = encoder.encode(password);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}

loginForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  console.log("in loginForm submit event :", e);
  const inputPassword: HTMLInputElement = document.getElementById(
    "password",
  )! as HTMLInputElement;
  const hashedPassword = await hashPasswordSha256(`${inputPassword.value}`);
  const inputHashedPassword: HTMLInputElement = document.getElementById(
    "hashed",
  )! as HTMLInputElement;
  inputHashedPassword.value = hashedPassword.toString();
  console.log("hashedPassword", hashedPassword);
  //const inputs = loginForm.elements;
  if (inputHashedPassword.value.length > 0 && inputPassword.value.length > 0) {
    const data = new FormData(loginForm);
    console.log("data", data);
    const serverProtocol = window.location.protocol;
    const url = `${serverProtocol}//${serverHost}/api/login`;
    const response = await fetch(url, {
      method: "post",
      body: data,
    });
    if (!response.ok) {
      const errorMessage = await response.text();
      msg.innerHTML = `<h4>${errorMessage}</h4>`;
    }
    const jsonResponse = await response.json();
    const niceToReadResponse = JSON.stringify(jsonResponse, null, 2);
    if ("token" in jsonResponse) {
      token = jsonResponse["token"];
      loginForm.style.display = "none";
      divMsg.style.display = "none";
      setupTerminal(
        document.querySelector<HTMLDivElement>("#terminal")!,
        token,
      );
    } else {
      msg.innerHTML = `<h4> token key not found in ${niceToReadResponse}</h4>`;
    }
    console.log(jsonResponse);
    msg.innerHTML = `response from server<pre>${niceToReadResponse}</pre>`;
  } else {
    msg.innerHTML = "<h4>Login and password values cannot be empty</h4>";
  }
});
