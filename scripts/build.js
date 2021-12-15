#!/use/bin/env node

"use strict";

const fs = require("fs-extra");
const path = require("path");

// const webpack = require("webpack");
// const config = require("../webpack.config");
// build().then((value) => {
//   // console.log(value);
// });

const appPublic = "./public";
const appBuild = "./build";

function build() {
  const compiler = webpack(config);
  return new Promise((resolve, reject) => {
    compiler.run((err, stats) => {
      if (err) {
        return reject(err);
      }
      return resolve(stats);
    });
  });
}

function copyPublicFolder() {
  fs.copySync(appPublic, appBuild, {
    dereference: true,
    filter: (file) => !file.endsWith("index.html"), // file !== "index.html",
  });
}

const importLocal = require("import-local");
const WebpackCLI = require("webpack-cli");

if (!process.env.WEBPACK_CLI_SKIP_IMPORT_LOCAL) {
  // Prefer the local installation of `webpack-cli`
  if (importLocal(__filename)) {
    return;
  }
}

process.title = "webpack";

const runCLI = async (args) => {
  const cli = new WebpackCLI();
  try {
    await cli.run(process.args);
  } catch (error) {
    cli.logger.error(error);
    process.exit(2);
  }
};

copyPublicFolder();
runCLI(process.args);
