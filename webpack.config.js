const CopyWebpackPlugin = require("copy-webpack-plugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const TerserPlugin = require("terser-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const CssMinimizerPlugin = require("css-minimizer-webpack-plugin");
const path = require("path");
const fs = require("fs");

const paths = {
  public: "./public",
  source: "./frontend",
  build: "./build",
};

const findIndex = () => {
  let p = path.join(paths.public, "index.html");
  if (fs.existsSync(p)) {
    return p;
  }
  return path.join(paths.source, "index.html");
};

module.exports = function (webpackEnv) {
  const isDev = webpackEnv.dev || false;
  const isProd = webpackEnv.prod || false;

  return {
    entry: {
      index: {
        import: path.resolve(__dirname, paths.source, "main.ts"),
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
      path: path.resolve(__dirname, paths.build),
    },

    optimization: {
      // runtimeChunk: "single",
      minimize: true,
      minimizer: [
        new TerserPlugin({
          terserOptions: {
            parse: { ecma: 8 },
            compress: {
              ecma: 5,
              warnings: false,
              comparisons: false,
            },
            output: {
              ecma: 5,
              comments: false,
              ascii_only: true,
            },
            sourceMap: true,
          },
        }),
        new CssMinimizerPlugin(),
      ],
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
          include: [path.resolve(__dirname, paths.source)],
        },
        {
          test: /\.s?css$/,
          use: [MiniCssExtractPlugin.loader, "css-loader", "sass-loader"],
        },
      ],
    },

    plugins: [
      new HtmlWebpackPlugin(
        Object.assign(
          {},
          {
            inject: true,
            template: findIndex(),
          },
          isProd
            ? {
                minify: {
                  collapseWhitespace: true,
                  removeComments: true,
                  keepClosingSlash: true,
                  removeRedundantAttributes: true,
                  removeStyleLinkTypeAttributes: true,
                  minifyJS: true,
                  minifyCSS: true,
                  minifyURLs: true,
                },
              }
            : undefined
        )
      ),
      new MiniCssExtractPlugin(),
      new CopyWebpackPlugin({ patterns: [{ from: paths.public }] }),
    ],
  };
};
