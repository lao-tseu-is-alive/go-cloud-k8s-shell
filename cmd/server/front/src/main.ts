import "./skeleton.css";
import "@xterm/xterm/css/xterm.css";

import { setupTerminal } from "./terminal.ts";

const html = `
<div class="container">
  <section class="header">
     <h5>goCloudK8sShell</h5>
  </section>  
  <div class="row">
    <div class="twelve columns">
      <div id="terminal"></div>
    </div>    
  </div>  
</div>

`;

document.querySelector<HTMLDivElement>("#app")!.innerHTML = html;
setupTerminal(document.querySelector<HTMLDivElement>("#terminal")!);
