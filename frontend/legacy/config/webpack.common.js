const path = require("path");

const TerserPlugin = require("terser-webpack-plugin");
const CssMinimizerPlugin = require("css-minimizer-webpack-plugin");

const resolve = (rootDir) /** @type import('webpack').ResolveOptions */ => {
  return {
    extensions: [".tsx", ".ts", ".jsx", ".js", ".css", ".svg"],
    alias: {
      "~/frontend": path.join(rootDir, "frontend", "legacy"),
      "~": rootDir,
    },
  };
};

const output = (paths, isProd) => {
  return {
    clean: isProd, // remove old files before build
    path: path.resolve(paths.rootDir, paths.build),
    // pathinfo: false,
    filename: isProd
      ? "static/js/[contenthash].js"
      : "static/js/[name].bundle.js",
    chunkFilename: isProd
      ? "static/js/[contenthash].chunk.js"
      : "static/js/[name].chunk.js",
    assetModuleFilename: isProd
      ? "static/a/[contenthash:16][ext]"
      : "static/a/[name][id][ext]",
  };
};

const optimization = (isProd, isCI) => {
  return {
    providedExports: true,
    usedExports: "global",
    concatenateModules: isProd,
    minimize: isProd && !isCI,
    minimizer: [
      new TerserPlugin({
        terserOptions: {
          compress: {
            ecma: 5,
            inline: true,
          },
          output: {
            ecma: 5,
            comments: false,
          },
          sourceMap: !isProd,
        },
      }),
      new CssMinimizerPlugin(),
    ],
    // Removing some optimizations during CI to make builds faster
    removeAvailableModules: isCI ? false : true,
    removeEmptyChunks: isCI ? false : true,
    splitChunks: isCI
      ? false
      : {
          chunks: "all",
          minSize: 20_000,
        },
  };
};

module.exports = {
  resolve,
  output,
  optimization,
};
