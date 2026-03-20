import { build } from "esbuild";
import { readFileSync } from "node:fs";

const { engines } = JSON.parse(readFileSync(new URL("./package.json", import.meta.url)));
const nodeVersionMatch = engines.node.match(/\d+/); // e.g. "24.x" or ">=24" → ["24"]
if (!nodeVersionMatch) {
  throw new Error(`Unable to determine Node.js major version from engines.node: ${engines.node}`);
}
const nodeVersion = nodeVersionMatch[0];

await build({
  entryPoints: ["src/index.ts"],
  bundle: true,
  platform: "node",
  target: `node${nodeVersion}`,
  format: "esm",
  outfile: "dist/index.js",
});
