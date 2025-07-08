"use strict";

import * as core from "@actions/core";
import * as tc from "@actions/tool-cache";
import * as os from "os";
import { Octokit } from "@octokit/rest";

if (require.main === module) {
  main().catch((err) => {
    console.error(err.stack);
    process.exit(1);
  });
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
            .getLatestRelease({
              owner: "nexthink-oss",
              repo: "ghup",
            })
            .then((res) => res.data.tag_name)
        : inputs.version;

    let ghupPath = tc.find("ghup", version);

    if (!ghupPath) {
      const platform = os.platform();
      let arch = os.arch();
      if (arch === "x64") {
        arch = "amd64";
      }

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
