const HtmlWebpackPlugin = require("html-webpack-plugin");
const path = require("path");
const fs = require("fs");

const public = "./public";
const source = "./frontend";
const build = "./build";

const findIndex = () => {
  let p = path.join(public, "index.html");
  if (fs.existsSync(p)) {
    return p;
  }
  return path.join(source, "index.html");
};

module.exports = function (webpackEnv) {
  const isDev = webpackEnv === "development";
  const isProd = webpackEnv === "production";
  return {
    entry: {
      index: {
        import: path.resolve(__dirname, source, "main.ts"),
      },
    },
    resolve: {
      extensions: [".tsx", ".ts", ".js"],
    },
    output: {
      // publicPath: 'public',
      filename: isProd
        ? "static/js/[name].[contenthash:8].bundle.js"
        : "static/js/[name].bundle.js",
      chunkFilename: isProd
        ? "static/js/[name].[contenthash:8].chunk.js"
        : "static/js/[name].chunk.js",
      path: path.resolve(__dirname, build),
    },
    optimization: {
      // runtimeChunk: "single",
      splitChunks: {
        chunks: "all",
        name: false,
      },
    },
    module: {
      rules: [
        {
          test: /\.tsx?$/,
          use: "ts-loader",
          exclude: /node_modules/,
          include: [path.resolve(__dirname, source)],
        },
      ],
    },
    plugins: [
      new HtmlWebpackPlugin(
        Object.assign(
          {},
          { inject: true, template: findIndex() },
          isProd
            ? {
                minify: {
                  removeComments: true,
                  removeRedundantAttributes: true,
                  minifyJS: true,
                  minifyCSS: true,
                  minifyURLs: true,
                },
              }
            : undefined
        )
      ),
    ],
  };
};
