import "./skeleton.css";
import "@xterm/xterm/css/xterm.css";
import "./style.css";
import { setupTerminal } from "./terminal.ts";
import { fetchAppInfo } from "./appInfo.ts";
import sha256 from 'crypto-js/sha256';


let token = null;
const serverHost = window.location.port === "5173" ? "localhost:9999" : window.location.host;
const serverProtocol = window.location.protocol;
const defaultJwtAuthUrl = "/login"
let jwtAuthUrl = defaultJwtAuthUrl;
const urlJwtLogin = (jwtUrl: string) => `${serverProtocol}//${serverHost}${jwtUrl}`;
const urlAppInfo= `${serverProtocol}//${serverHost}/goAppInfo`;

const html = `
<div class="container">
  <section class="header">
     <h6 id="appInfoHeading">Loading app info...</h6>
  </section>  
  <form method="post" action="${urlJwtLogin(jwtAuthUrl)}" class="u-full-width" id="loginForm">
        <div class="row">
            <div class="six columns">
                <label for="login">Login:</label><br/>
                <input id="login" type="text" name="login" autocomplete="username" placeholder="Enter your login here" 
                value="" class="u-full-width">
            </div>
            <div class="six columns">
                <label for="password">Password:</label><br/>
                <input id="password" type="password" autocomplete="current-password" name="password"
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
      <div id="terminal" class="my-terminal"></div>
    </div>    
  </div>  
</div>

`;


/*async function hashPasswordSha256(password: string) {
  const encoder = new TextEncoder();
  const data = encoder.encode(password);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}*/




document.addEventListener("DOMContentLoaded", async () => {
  document.querySelector<HTMLDivElement>("#app")!.innerHTML = html;
  const loginForm: HTMLFormElement = document.getElementById("loginForm")! as HTMLFormElement;
  const divMsg: HTMLDivElement =  document.querySelector<HTMLDivElement>("#divMsg")!;
  async function loginSubmitHandler(e: Event) {
    const msg = document.getElementById("msg")!;
    e.preventDefault();
    console.log("in loginForm submit event :", e);
    const inputPassword: HTMLInputElement = document.getElementById(
      "password",
    )! as HTMLInputElement;
    const hashedPassword = sha256(inputPassword.value);
    const inputHashedPassword: HTMLInputElement = document.getElementById(
      "hashed",
    )! as HTMLInputElement;
    inputHashedPassword.value = hashedPassword.toString();
    console.log("hashedPassword", inputHashedPassword.value);
    //const inputs = loginForm.elements;
    if (inputHashedPassword.value.length > 0 && inputPassword.value.length > 0) {
      const data = new FormData(loginForm);
      console.log("data", data);

      const response = await fetch(urlJwtLogin(jwtAuthUrl), {
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
  }





  const appInfoHeading = document.getElementById("appInfoHeading") as HTMLHeadingElement | null;
  loginForm.addEventListener("submit", loginSubmitHandler);
  if (!appInfoHeading || !loginForm) {
    console.error("Required DOM elements (appInfoHeading or loginForm) not found.");
    return;
  }

  try {
    const appData = await fetchAppInfo(urlAppInfo); // Call the generic function
    // Now, use the returned appData to update the DOM
    if (appData.app && appData.version) {
      appInfoHeading.textContent = `${appData.app} - v${appData.version}`;
    } else {
      appInfoHeading.textContent = "App Info Not Available";
      console.error("App name or version not found in fetched data.");
    }

    if (appData.authUrl) {
      jwtAuthUrl = appData.authUrl;
      console.log(`Login jwtAuthUrl set to: ${jwtAuthUrl}`);
    } else {
      console.error("authUrl not found in fetched data.");
      jwtAuthUrl = defaultJwtAuthUrl; // Fallback
    }
  } catch (error) {
    console.error("Error during app initialization:", error);
    appInfoHeading.textContent = "Failed to Load App Info";
    jwtAuthUrl = defaultJwtAuthUrl; // Fallback
  }
});
