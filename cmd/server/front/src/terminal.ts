import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SerializeAddon } from "@xterm/addon-serialize";
import { Unicode11Addon  } from "@xterm/addon-unicode11";
import { WebLinksAddon } from "@xterm/addon-web-links";
import {AttachAddon} from "@xterm/addon-attach";
import { BGred, yellow, bright, reset} from "./consoleColors.ts";

export function setupTerminal(element: HTMLDivElement) {
  const protocol = (location.protocol === "https:") ? "wss://" : "ws://";
  //const url = protocol + location.host + "/goshell"
  const url = protocol + "localhost:9999/goshell"
  //const url = protocol + "localhost:8376/xterm.js"
  const ws = new WebSocket(url);
  const term = new Terminal({
    allowProposedApi: true,
    cursorBlink: true,
    screenReaderMode: true,
    cols: 128,
  });
  const fitAddon = new FitAddon();
  const unicode11Addon = new Unicode11Addon();
  const serializeAddon = new SerializeAddon();
  const webLinksAddon = new WebLinksAddon();
  const attachAddon = new AttachAddon(ws, { bidirectional: true });
  term.loadAddon(fitAddon);
  term.loadAddon(unicode11Addon);
  term.loadAddon(serializeAddon);
  term.loadAddon(webLinksAddon);
  term.unicode.activeVersion = '11';
  // activate the new version
  term.open(element);
  fitAddon.fit();
  /*term.write("Hello from \x1B[1;3;31mxterm.js\x1B[0m $ World!", () => {
    console.log(serializeAddon.serialize());
  });*/
  //element.addEventListener('click', () => setCounter(counter + 1))
  ws.onopen = function() {
    term.loadAddon(attachAddon);
    term.focus();
    setTimeout(function() {fitAddon.fit()});
    term.onResize(function(event) {
      const rows = event.rows;
      const cols = event.cols;
      const size = JSON.stringify({cols: cols, rows: rows + 1});
      const send = new TextEncoder().encode("\x01" + size);
      console.log('resizing to', size);
      ws.send(send);
    });
    term.onTitleChange(function(event) {
      console.log(event);
    });
    window.onresize = function() {
      fitAddon.fit();
    };
  };
  ws.onerror = function(event) {
    console.log("ws.onerror event:", event);
    term.write(`💥💥 ${BGred} ${bright} ${yellow} Websocket ERROR EVENT, disconnected from server... ${reset} \r\n`, () => {
      console.log(serializeAddon.serialize());
    });
    console.log(event);
  }
  ws.onclose = function(event) {
    console.log("ws.onclose event:", event);
    term.write(`\r\n💥💥 ${BGred} ${bright} ${yellow} Websocket CLOSE EVENT, disconnected from server... ${reset} \r\n`, () => {
      console.log(serializeAddon.serialize());
    });
    console.log(event);
  }
}
