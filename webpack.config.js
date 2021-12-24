const CopyWebpackPlugin = require("copy-webpack-plugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const TerserPlugin = require("terser-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const CssMinimizerPlugin = require("css-minimizer-webpack-plugin");
const SitemapPlugin = require("sitemap-webpack-plugin").default;
const CompressionPlugin = require("compression-webpack-plugin");
const path = require("path");
const build = require("./scripts/build");

// Used for build-time template parameters
const site = require("./site");

const paths = {
  public: "./public",
  source: "./frontend",
  build: "./build",
};

paths.favicon = path.join(paths.public, "favicon.ico");

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
  return {
    from: path.join(paths.public, name),
    to: name,
  };
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

module.exports = function (webpackEnv) {
  const isDev = webpackEnv.dev || false;
  const isProd = webpackEnv.prod || false;

  return {
    entry: {
      main: {
        import: path.resolve(__dirname, paths.source, "main.ts"),
      },
      remora: {
        import: path.resolve(__dirname, paths.source, "remora.ts"),
      },
      harry_y_tanya: {
        import: path.resolve(__dirname, paths.source, "pages/harry-y-tanya.ts"),
      },
    },

    resolve: {
      extensions: [".tsx", ".ts", ".js"],
    },

    output: {
      clean: isProd, // remove old files before build
      path: path.resolve(__dirname, paths.build),
      filename: isProd
        ? "static/js/[name].[contenthash:16].bundle.js"
        : "static/js/[name].bundle.js",
      chunkFilename: isProd
        ? "static/js/[name].[contenthash:16].chunk.js"
        : "static/js/[name].chunk.js",
      assetModuleFilename: "static/a/[hash:4].[id][ext][query][fragment]",
    },

    optimization: {
      runtimeChunk: "single",
      concatenateModules: true,
      minimize: true,
      //runtimeChunk: { name: "runtime" },
      minimizer: [
        new TerserPlugin({
          terserOptions: {
            parse: { ecma: 8 },
            compress: {
              ecma: 5,
              warnings: true,
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
          include: [path.resolve(__dirname, paths.source)],
        },
        {
          test: /\.s?css$/,
          use: [
            // for importing css in js/ts and injecting it into the DOM
            "style-loader",
            // for @import in css
            "css-loader",
            //MiniCssExtractPlugin.loader,
          ],
          include: [path.resolve(__dirname, paths.source)],
        },
        {
          // Embed these right into the html
          test: /\.(gif|svg)$/i,
          type: "asset/inline",
        },
        {
          // Fonts
          test: /\.(woff(2)?|ttf|eot)(\?v=\d+\.\d+\.\d+)?$/,
          type: "asset/inline",
        },
        {
          // Load these as static resources
          test: /\.(pdf|jpe?g|png)$/i,
          type: "asset/resource",
        },
      ],
    },

    plugins: [
      new HtmlWebpackPlugin(
        Object.assign(
          {},
          {
            template: path.join(paths.source, "index.html"),
            templateParameters: site.pages["index"],
            meta: build.metaTags(site.pages["index"]),
            chunks: ["main"],
            favicon: paths.favicon,
          },
          isProd ? { minify: htmlMinify } : { cache: true }
        )
      ),
      new HtmlWebpackPlugin(
        Object.assign(
          {},
          // {
          //   template: path.join(paths.source, "pages/remora.html"),
          //   filename: "pages/remora.html",
          //   templateParameters: site.pages["remora"],
          //   chunks: ["remora"],
          //   meta: build.metaTags(site.pages["remora"]),
          //   favicon: paths.favicon,
          // },
          // { favicon: paths.favicon },
          build.page(paths, "remora", site.pages["remora"]),
          isProd ? { minify: htmlMinify } : { cache: true }
        )
      ),
      new HtmlWebpackPlugin(
        Object.assign(
          {},
          build.page(paths, "harry-y-tanya", site.pages["harrytanya"]),
          isProd ? { minify: htmlMinify } : { cache: true }
        )
      ),

      new CompressionPlugin({
        deleteOriginalAssets: true,
        filename: "[path][base]",
        test: isProd ? /index\.html/ : /^$/,
        exclude: [
          /sitemap\.xml$/,
          /robots\.txt$/,
          /favicon\.ico/,
          /.*\.asc$/, // all public keys
        ],
      }),
      new CopyWebpackPlugin({
        patterns: [
          // Copy over the legacy site... just for the lols
          copy("static/js/bootstrap.min.js"),
          copy("static/js/popper.min.js"),
          copy("static/js/jquery-3.4.1.min.js"),
          copy("static/js/home.js"),
          copy("static/css/bootstrap.min.css"),
          copy("static/css/animate.min.css"),
          copy("static/css/base.css"),
          copy("static/css/home.css"),
          copy("static/img/linkedin.svg"),
          copy("static/img/github.svg"),
          copy("static/img/1125x1500/me_sm.jpg"),
          // I actually need these
          copy("static/files"),
          {
            // Harry's Preview Image
            from: path.join(paths.source, "img/goofy.jpg"),
            to: path.resolve(__dirname, paths.build, "static/img/goofy.jpg"),
          },
          { from: path.join(paths.public, "robots.txt") },
          { from: path.join(paths.public, "pub.asc") },
        ],
      }),
      new SitemapPlugin({
        base: "https://harrybrwn.com",
        paths: sitemap,
        options: { skipgzip: false },
      }),
    ],
  };
};
