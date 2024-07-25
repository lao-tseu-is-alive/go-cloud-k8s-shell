import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SerializeAddon } from "@xterm/addon-serialize";
import { Unicode11Addon  } from "@xterm/addon-unicode11";
import { WebLinksAddon } from "@xterm/addon-web-links";
import {AttachAddon} from "@xterm/addon-attach";

export function setupTerminal(element: HTMLDivElement) {
  const protocol = (location.protocol === "https:") ? "wss://" : "ws://";
  const url = protocol + location.host + "/goshell"
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
  const attachAddon = new AttachAddon(ws);
  term.loadAddon(fitAddon);
  term.loadAddon(unicode11Addon);
  term.loadAddon(serializeAddon);
  term.loadAddon(webLinksAddon);
  term.unicode.activeVersion = '11';
  // activate the new version
  term.open(element);
  fitAddon.fit();
  term.write("Hello from \x1B[1;3;31mxterm.js\x1B[0m $ World!", () => {
    console.log(serializeAddon.serialize());
  });
  //element.addEventListener('click', () => setCounter(counter + 1))
  ws.onopen = function() {
    term.write("ws \x1B[1;3;31mopen\x1B[0m $ event ", () => {
      console.log(serializeAddon.serialize());
    });
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
    term.write("ws \x1B[1;3;31merror\x1B[0m $ event, disconnected from server... ", () => {
      console.log(serializeAddon.serialize());
    });
    console.log(event);
  }
}
