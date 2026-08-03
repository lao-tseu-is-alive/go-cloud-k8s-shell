import { join } from "node:path";

const rootDir = process.cwd();
const distDir = join(rootDir, "dist");
const port = Number(Bun.env.PORT ?? "4173");

function contentType(pathname: string) {
  if (pathname.endsWith(".css")) return "text/css";
  if (pathname.endsWith(".html")) return "text/html";
  if (pathname.endsWith(".js")) return "text/javascript";
  if (pathname.endsWith(".svg")) return "image/svg+xml";
  return "application/octet-stream";
}

Bun.serve({
  port,
  async fetch(request) {
    const url = new URL(request.url);
    const pathname = url.pathname === "/" ? "/index.html" : url.pathname;
    const file = Bun.file(join(distDir, pathname));

    if (!(await file.exists())) {
      return new Response("Not found", { status: 404 });
    }

    return new Response(file, {
      headers: {
        "content-type": contentType(pathname),
      },
    });
  },
});

console.log(`Bun preview server running at http://localhost:${port}/`);
