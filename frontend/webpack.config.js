"use strict";

const CopyWebpackPlugin = require("copy-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const HTMLInlineCSSWebpackPlugin =
  require("html-inline-css-webpack-plugin").default;
const SitemapPlugin = require("sitemap-webpack-plugin").default;
const ForkTsCheckerWebpackPlugin = require("fork-ts-checker-webpack-plugin");
const fs = require("fs");
const hjson = require("hjson");

const path = require("path");
const build = require("./config/build");
const common = require("./config/webpack.common");

const tsconfig = hjson.parse(
  fs.readFileSync("./tsconfig.json", { encoding: "ascii" })
);

// Used for build-time template parameters
const site = require("./site");

const REPO_ROOT = path.resolve(__dirname, "..");

const paths = {
  rootDir: path.resolve(__dirname),
  source: "./",
  public: "./public",
  build: path.resolve(REPO_ROOT, "build", "harrybrwn.com"),
  favicon: "public/favicon.ico",
  cache: path.resolve(REPO_ROOT, ".cache/build"),
};

const BABEL_CONFIG = path.resolve(
  paths.rootDir,
  paths.source,
  "config",
  "babel.config.js"
);

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
  {
    path: "/bookmarks",
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
      from: path.join(paths.rootDir, paths.public, name),
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

const makeCopyPlugin = (builder) => {
  return new CopyWebpackPlugin({
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
      // copy("static/files"),
      {
        from: path.join(builder.paths.rootDir, builder.paths.source, "files"),
        to: path.join(builder.paths.build, "static/files"),
      },
      {
        // Harry's OpenGraph Preview Image
        from: path.join(
          builder.paths.rootDir,
          builder.paths.source,
          "img/goofy.jpg"
        ),
        to: path.resolve(builder.paths.build, "static/img/goofy.jpg"),
      },
      { from: path.join(builder.paths.public, "robots.txt") },
      { from: path.join(builder.paths.public, "pub.asc") },
      { from: path.join(builder.paths.source, "manifest.json") },
    ],
  });
};

module.exports = function (webpackEnv) {
  console.log(webpackEnv);
  const isProd = webpackEnv.prod || false;
  const isCI = webpackEnv.ci || false;
  const isWatch = webpackEnv.WEBPACK_WATCH || webpackEnv.WEBPACK_SERVE || false;
  const builder = new build.Builder({
    paths,
    site,
    isProd,
    htmlMinify,
  });
  const embedCSS = false;

  let plugins = [
    new ForkTsCheckerWebpackPlugin(), // Typechecking in a different process
    new MiniCssExtractPlugin({
      filename: builder.isProd
        ? "static/css/[contenthash:16].css"
        : "static/css/[name].[id].css",
    }),
    makeCopyPlugin(builder),
    new SitemapPlugin({
      base: `https://${builder.site.domain}`,
      paths: sitemap,
      options: { skipgzip: false },
    }),
  ];
  if (!isWatch && embedCSS) {
    plugins.push(new HTMLInlineCSSWebpackPlugin());
  }
  plugins.push(
    builder.html("index", { pageDir: ".", chunks: ["main"] }),
    builder.html("remora"),
    builder.html("admin"),
    builder.html("harry_y_tanya"),
    builder.html("games"),
    builder.html("404", { noChunks: true, filename: "404.html" }),
    builder.html("50x", { noChunks: true, filename: "50x.html" }),
    builder.html("invite"),
    builder.html("chatroom"),
    builder.html("invite_email", { noChunks: true }),
    builder.html("bookmarks")
  );

  for (const key in site.pages) {
    // TODO generate parts of the config with this
  }

  const entryImport = (name) => {
    let p = path.resolve(paths.rootDir, paths.source, name);
    if (!fs.existsSync(p)) {
      console.error("path", p, "does not exist");
    }
    return p;
  };
  return {
    entry: {
      main: { import: entryImport("main.ts") },
      remora: { import: entryImport("pages/remora.ts") },
      harry_y_tanya: { import: entryImport("pages/harry_y_tanya.ts") },
      admin: { import: entryImport("pages/admin/admin.ts") },
      games: { import: entryImport("pages/games.ts") },
      chatroom: { import: entryImport("pages/chatroom/chatroom.ts") },
      invite: { import: entryImport("pages/invite.ts") },
      bookmarks: { import: entryImport("pages/bookmarks/bookmarks.ts") },
    },

    devtool: builder.isProd ? undefined : "inline-source-map",
    resolve: common.resolve(path.resolve(REPO_ROOT)),
    output: common.output(paths, builder.isProd),
    optimization: common.optimization(isProd, isCI),

    module: {
      rules: [
        {
          test: /\.(js|ts)x?$/,
          use: {
            loader: require.resolve("babel-loader"),
            options: {
              cacheDirectory: path.resolve(paths.rootDir, paths.cache, "babel"),
              cacheCompression: false,
              configFile: BABEL_CONFIG,
              babelrc: false, // ignore any .babelrc file
            },
          },
          include: [path.resolve(paths.rootDir, paths.source)],
        },
        {
          test: /\.s?css$/,
          use: [
            MiniCssExtractPlugin.loader,
            // for @import in css
            require.resolve("css-loader"),
          ],
          include: [path.resolve(paths.rootDir, paths.source)],
        },
        {
          // Embed svg into the html
          test: /\.(svg)$/i,
          //type: isProd ? "asset/inline" : "asset/resource",
          type: "asset/inline",
        },
        {
          test: /\.(gif)$/i,
          type: "asset/resource",
        },
        { test: /stars-compressed\.webp/i, type: "asset/inline" },
        {
          // Fonts
          test: /\.(woff(2)?|ttf|eot)(\?v=\d+\.\d+\.\d+)?$/,
          type: "asset/resource", // inline fonts make parsing really slow
        },
        {
          // Load these as static resources
          test: /\.(pdf|jpe?g|png)$/i,
          type: "asset/resource",
        },
      ],
    },

    plugins: plugins,

    cache: {
      type: "filesystem",
      cacheDirectory: path.resolve(REPO_ROOT, paths.cache, "webpack"),
      store: "pack",
      buildDependencies: {
        // This makes all dependencies of this file - build dependencies
        //config: [__filename, path.resolve("./site.js")],
      },
    },
    devServer: {
      port: 9000,
    },
  };
};
