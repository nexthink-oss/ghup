import { build } from "esbuild";
import { readFileSync, writeFileSync } from "node:fs";

const { engines } = JSON.parse(readFileSync(new URL("./package.json", import.meta.url)));
const nodeVersion = engines.node.replace(/\D.*/, ""); // "24.x" → "24"

const actionYaml = new URL("./action.yaml", import.meta.url);
writeFileSync(actionYaml, readFileSync(actionYaml, "utf8").replace(/using: "node\d+"/, `using: "node${nodeVersion}"`));

await build({
  entryPoints: ["src/index.ts"],
  bundle: true,
  platform: "node",
  target: `node${nodeVersion}`,
  outfile: "dist/index.js",
});
