"use strict";

const CopyWebpackPlugin = require("copy-webpack-plugin");
const TerserPlugin = require("terser-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const CssMinimizerPlugin = require("css-minimizer-webpack-plugin");
const HTMLInlineCSSWebpackPlugin =
  require("html-inline-css-webpack-plugin").default;
const SitemapPlugin = require("sitemap-webpack-plugin").default;
const path = require("path");
const build = require("./scripts/build");

// Used for build-time template parameters
const site = require("./site");

const paths = {
  public: "./public",
  source: "./frontend",
  build: "./build",
  favicon: "./public/favicon.ico",
};

const sitemap = [
  {
    path: "/",
  },
  {
    path: "/~harry",
  },
  {
    path: "/static/files/HarrisonBrown.pdf",
  },
];

const copy = (name) => {
  if (Array.isArray(name)) {
    let copies = [];
    for (let n of name) {
      copies.push(copy(n));
    }
    return copies;
  } else {
    return {
      from: path.join(paths.public, name),
      to: name,
    };
  }
};

const htmlMinify = {
  collapseWhitespace: true,
  removeComments: true,
  keepClosingSlash: true,
  removeRedundantAttributes: true,
  removeStyleLinkTypeAttributes: true,
  minifyJS: true,
  minifyCSS: true,
  minifyURLs: true,
};

const plugins = (builder) => {
  let plugins = [
    new MiniCssExtractPlugin({
      filename: builder.isProd
        ? "static/css/[name].[hash:8].css"
        : "static/css/[name].css",
    }),
    builder.page("index", { pageDir: ".", chunks: ["main"] }),
    builder.page("remora"),
    builder.page("admin"),
    builder.page("404", { noChunks: true }),
    builder.page("harry_y_tanya"),
    builder.page("games"),

    new CopyWebpackPlugin({
      patterns: [
        // Copy over the legacy site... just for the lols
        ...copy(
          [
            "js/bootstrap.min.js",
            "js/popper.min.js",
            "js/jquery-3.4.1.min.js",
            "js/home.js",
            "css/bootstrap.min.css",
            "css/animate.min.css",
            "css/base.css",
            "css/home.css",
            "img/linkedin.svg",
            "img/github.svg",
            "img/1125x1500/me_sm.jpg",
          ].map((v) => path.join("static", v))
        ),
        copy("static/files"),
        {
          // Harry's OpenGraph Preview Image
          from: path.join(builder.paths.source, "img/goofy.jpg"),
          to: path.resolve(
            __dirname,
            builder.paths.build,
            "static/img/goofy.jpg"
          ),
        },
        { from: path.join(builder.paths.public, "robots.txt") },
        { from: path.join(builder.paths.public, "pub.asc") },
        { from: path.join(builder.paths.source, "manifest.json") },
      ],
    }),
    new SitemapPlugin({
      base: `https://${builder.site.domain}`,
      paths: sitemap,
      options: { skipgzip: false },
    }),
  ];
  // if (builder.isProd) {
  //   plugins.push(new HTMLInlineCSSWebpackPlugin());
  // }
  return plugins;
};

module.exports = function (webpackEnv) {
  const isProd = webpackEnv.prod || false;
  const builder = new build.Builder({
    paths,
    site,
    isProd,
    htmlMinify,
  });

  for (const key in site.pages) {
    // TODO generate parts of the config with this
  }

  return {
    entry: {
      main: {
        import: path.resolve(__dirname, paths.source, "main.ts"),
      },
      remora: {
        import: path.resolve(__dirname, paths.source, "pages/remora.ts"),
      },
      harry_y_tanya: {
        import: path.resolve(__dirname, paths.source, "pages/harry_y_tanya.ts"),
      },
      admin: {
        import: path.resolve(__dirname, paths.source, "pages/admin.ts"),
      },
      games: {
        import: path.resolve(__dirname, paths.source, "pages/games.ts"),
      },
    },

    resolve: {
      extensions: [".tsx", ".ts", ".js", ".css"],
      alias: {
        "@harrybrwn.com": path.resolve(__dirname, "./") + "/",
        "~": __dirname,
      },
    },

    output: {
      clean: isProd, // remove old files before build
      path: path.resolve(__dirname, paths.build),
      filename: isProd
        ? "static/js/[name].[contenthash].js"
        : "static/js/[name].bundle.js",
      chunkFilename: isProd
        ? "static/js/[name].[contenthash].chunk.js"
        : "static/js/[name].chunk.js",
      assetModuleFilename: "static/a/[hash:4].[id][ext][query][fragment]",
    },

    optimization: {
      concatenateModules: isProd,
      providedExports: true,
      usedExports: "global",
      minimize: isProd,
      minimizer: [
        new TerserPlugin({
          terserOptions: {
            compress: {
              ecma: 5,
            },
            output: {
              ecma: 5,
              comments: false,
            },
            sourceMap: true,
          },
        }),
        new CssMinimizerPlugin(),
      ],
    },

    module: {
      rules: [
        {
          test: /\.tsx?$/,
          use: "ts-loader",
          include: [path.resolve(__dirname, paths.source)],
        },
        {
          test: /\.s?css$/,
          use: [
            MiniCssExtractPlugin.loader,
            // for @import in css
            "css-loader",
          ],
          include: [path.resolve(__dirname, paths.source)],
        },
        {
          // Embed these right into the html
          test: /\.(gif|svg)$/i,
          type: isProd ? "asset/inline" : "asset/resource",
        },
        {
          // Fonts
          test: /\.(woff(2)?|ttf|eot)(\?v=\d+\.\d+\.\d+)?$/,
          // type: "asset/inline",
          type: "asset/resource",
        },
        {
          // Load these as static resources
          test: /\.(pdf|jpe?g|png)$/i,
          type: "asset/resource",
        },
      ],
    },

    plugins: plugins(builder),
  };
};
