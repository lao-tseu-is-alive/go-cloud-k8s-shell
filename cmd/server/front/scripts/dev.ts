import { cp, mkdir, readdir, rm, stat } from "node:fs/promises";
import { join, relative } from "node:path";

const rootDir = process.cwd();
const devDir = join(rootDir, ".bun-dev");
const assetsDir = join(devDir, "assets");
const port = Number(Bun.env.PORT ?? "5173");

let buildPromise: Promise<void> | null = null;
let lastBuild = 0;

async function build() {
  await rm(devDir, { recursive: true, force: true });
  await mkdir(assetsDir, { recursive: true });
  await cp(join(rootDir, "public"), devDir, { recursive: true });

  const result = await Bun.build({
    entrypoints: [join(rootDir, "src/main.ts")],
    outdir: assetsDir,
    target: "browser",
    sourcemap: "inline",
    naming: {
      entry: "[name].[ext]",
      asset: "[name].[ext]",
    },
  });

  if (!result.success) {
    throw new Error(result.logs.map(String).join("\n"));
  }

  const script = result.outputs.find((output) => output.kind === "entry-point");
  const stylesheet = result.outputs.find((output) =>
    output.type.startsWith("text/css"),
  );

  if (!script) {
    throw new Error("Bun build did not emit a JavaScript entry point.");
  }

  const assetPath = (path: string) =>
    `/${relative(devDir, path).replace(/\\/g, "/")}`;
  const index = `<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <link rel="icon" type="image/svg+xml" href="/goeland_io.svg" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>goCloudK8sShell</title>
  ${stylesheet ? `<link rel="stylesheet" href="${assetPath(stylesheet.path)}" />` : ""}
</head>
<body>
<div id="app"></div>
<script type="module" src="${assetPath(script.path)}"></script>
</body>
</html>
`;

  await Bun.write(join(devDir, "index.html"), index);
  lastBuild = Date.now();
}

async function ensureBuilt() {
  const latestSourceChange = Math.max(
    await latestModifiedTime(join(rootDir, "src")),
    await latestModifiedTime(join(rootDir, "public")),
  );

  if (!buildPromise || latestSourceChange > lastBuild) {
    buildPromise = build();
  }
  await buildPromise;
}

async function latestModifiedTime(path: string): Promise<number> {
  const info = await stat(path);

  if (!info.isDirectory()) {
    return info.mtimeMs;
  }

  const entries = await readdir(path, { withFileTypes: true });
  const times = await Promise.all(
    entries.map((entry) => latestModifiedTime(join(path, entry.name))),
  );

  return Math.max(info.mtimeMs, ...times);
}

function contentType(pathname: string) {
  if (pathname.endsWith(".css")) return "text/css";
  if (pathname.endsWith(".html")) return "text/html";
  if (pathname.endsWith(".js")) return "text/javascript";
  if (pathname.endsWith(".svg")) return "image/svg+xml";
  return "application/octet-stream";
}

await ensureBuilt();

Bun.serve({
  port,
  async fetch(request) {
    try {
      await ensureBuilt();
      const url = new URL(request.url);
      const pathname = url.pathname === "/" ? "/index.html" : url.pathname;
      const file = Bun.file(join(devDir, pathname));

      if (!(await file.exists())) {
        return new Response("Not found", { status: 404 });
      }

      return new Response(file, {
        headers: {
          "content-type": contentType(pathname),
          "cache-control": "no-store",
        },
      });
    } catch (error) {
      console.error(error);
      return new Response("Internal dev server error", { status: 500 });
    }
  },
});

console.log(`Bun dev server running at http://localhost:${port}/`);
