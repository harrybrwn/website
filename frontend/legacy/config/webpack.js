"use strict";

const webpack = require("webpack");
const configFactory = require("../webpack.config");

function build() {
  let config = configFactory({
    WEBPACK_BUNDLE: true,
    WEBPACK_BUILD: true,
    prod: true,
  });
  webpack.validate(config);

  let compiler = webpack(config);
  compiler.run((err, stats) => {
    if (err) {
      console.error(err);
      process.exit(1);
    }

    if (stats.hasErrors()) {
      console.error(stats.compilation.errors);
      process.exit(1);
    }
  });
}

build();
