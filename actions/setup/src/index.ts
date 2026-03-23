import * as core from "@actions/core";
import * as tc from "@actions/tool-cache";
import * as os from "node:os";
import { Octokit } from "@octokit/rest";

try {
  await main();
} catch (err) {
  if (err instanceof Error && err.stack) {
    console.error(err.stack);
  } else {
    console.error(err);
  }
  process.exit(1);
}

async function main(): Promise<void> {
  try {
    const inputs = {
      version: core.getInput("version") || "latest",
      token: core.getInput("token"),
    };

    const github = new Octokit({
      auth: inputs.token || undefined,
    });

    const version =
      inputs.version === "latest"
        ? await github.repos
            .getLatestRelease({ owner: "nexthink-oss", repo: "ghup" })
            .then((res) => res.data.tag_name)
        : inputs.version;

    let ghupPath = tc.find("ghup", version);

    if (!ghupPath) {
      const platform = os.platform();
      const rawArch = os.arch();
      const arch: string = rawArch === "x64" ? "amd64" : rawArch;

      const ghupUrl = `https://github.com/nexthink-oss/ghup/releases/download/${version}/ghup_${version.slice(1)}_${platform}_${arch}.zip`;
      const ghupZip = await tc.downloadTool(ghupUrl, undefined, inputs.token ? `token ${inputs.token}` : undefined);
      const extractPath = await tc.extractZip(ghupZip);
      ghupPath = await tc.cacheFile(
        `${extractPath}/ghup`,
        "ghup",
        "ghup",
        version,
      );
    }

    core.addPath(ghupPath);
    core.setOutput("version", version);
    core.info(`Finished setting up ghup ${version}`);
  } catch (err) {
    core.setFailed(`Action failed with error ${err}`);
  }
}
