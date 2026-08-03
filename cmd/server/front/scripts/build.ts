import { cp, mkdir, rm, writeFile } from "node:fs/promises";
import { join, relative } from "node:path";

const rootDir = process.cwd();
const distDir = join(rootDir, "dist");
const assetsDir = join(distDir, "assets");

await rm(distDir, { recursive: true, force: true });
await mkdir(assetsDir, { recursive: true });
await cp(join(rootDir, "public"), distDir, { recursive: true });

const result = await Bun.build({
  entrypoints: [join(rootDir, "src/main.ts")],
  outdir: assetsDir,
  target: "browser",
  minify: true,
  naming: {
    entry: "[name]-[hash].[ext]",
    asset: "[name]-[hash].[ext]",
  },
});

if (!result.success) {
  for (const log of result.logs) {
    console.error(log);
  }
  process.exit(1);
}

const script = result.outputs.find((output) => output.kind === "entry-point");
const stylesheet = result.outputs.find((output) =>
  output.type.startsWith("text/css"),
);

if (!script) {
  console.error("Bun build did not emit a JavaScript entry point.");
  process.exit(1);
}

const assetPath = (path: string) =>
  `/${relative(distDir, path).replace(/\\/g, "/")}`;

await writeFile(
  join(distDir, "index.html"),
  `<!doctype html>
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
`,
);
