const CopyWebpackPlugin = require("copy-webpack-plugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const TerserPlugin = require("terser-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const CssMinimizerPlugin = require("css-minimizer-webpack-plugin");
const SitemapPlugin = require("sitemap-webpack-plugin").default;
const CompressionPlugin = require("compression-webpack-plugin");
const package = require("./package.json");
const path = require("path");
const fs = require("fs");

// Used for build-time template parameters
const site = require("./site");

const paths = {
  public: "./public",
  source: "./frontend",
  build: "./build",
};

const sitemap = [
  {
    path: "/old/",
  },
  {
    path: "/static/files/HarrisonBrown.pdf",
  },
];

const findIndex = () => {
  let p = path.join(paths.public, "index.html");
  if (fs.existsSync(p)) {
    return p;
  }
  return path.join(paths.source, "index.html");
};

const copy = (name) => {
  return {
    from: path.join(paths.public, name),
    to: name,
  };
};

const fileCompressionLoader = {
  loader: "image-webpack-loader",
  options: {
    disable: false, // webpack@2.x and newer
    // Compress jpeg images
    mozjpeg: {
      progressive: true,
    },
    // Compress gif
    gifsicle: {
      interlaced: true,
    },
  },
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

class InjectImagesPlugin {
  apply(compiler) {
    const name = "InjectImagesPlugin";
    compiler.hooks.compilation.tap(name, (compilation) => {
      console.log("starting compiler hook");
      HtmlWebpackPlugin.getHooks(compilation).alterAssetTags.tapAsync(
        name,
        (data, cb) => {
          console.log(data.assetTags);
          cb(null, data);
        }
      );

      HtmlWebpackPlugin.getHooks(compilation).beforeEmit.tapAsync(
        name,
        (data, cb) => {
          console.log(data);
          cb(null, data);
        }
      );
    });
  }
}

module.exports = function (webpackEnv) {
  const isDev = webpackEnv.dev || false;
  const isProd = webpackEnv.prod || false;
  console.log(MiniCssExtractPlugin.loader);

  return {
    entry: {
      index: {
        import: path.resolve(__dirname, paths.source, "main.ts"),
      },
      remora: {
        import: path.resolve(__dirname, paths.source, "remora.ts"),
      },
      // tester: {import: path.resolve(__dirname, paths.source, "main.js")},
    },
    resolve: {
      extensions: [".tsx", ".ts", ".js"],
    },
    output: {
      clean: isProd, // remove old files before build
      // publicPath: 'public',
      path: path.resolve(__dirname, paths.build),
      filename: isProd
        ? "static/js/[name].[contenthash:8].bundle.js"
        : "static/js/[name].bundle.js",
      chunkFilename: isProd
        ? "static/js/[name].[contenthash:8].chunk.js"
        : "static/js/[name].chunk.js",
      assetModuleFilename: "static/a/[hash:4].[id][ext][query][fragment]",
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
          include: [path.resolve(__dirname, paths.source)],
        },
        {
          test: /\.s?css$/,
          use: [
            //
            // MiniCssExtractPlugin.loader,
            "style-loader",
            "css-loader",
          ],
          include: [path.resolve(__dirname, paths.source)],
        },
        {
          test: /\.gif$/,
          // type: "asset/resource",
          type: "asset/inline",
        },
        {
          test: /\.(g_if|png|jpe?g|svg)$/i,
          use: [
            fileCompressionLoader,
            // "url-loader",
            {
              loader: "file-loader",
              options: { name: "static/img/[name].[contenthash].[ext]" },
              // options: { name: "[name].[contenthash].[ext]" },
            },
          ],
          include: [path.resolve(__dirname, paths.source)],
        },
        {
          test: /\.(woff(2)?|ttf|eot|svg)(\?v=\d+\.\d+\.\d+)?$/,
          // type: "asset/resource",
          type: "asset/inline",
        },
        {
          test: /\.pdf$/,
          type: "asset/resource",
        },
      ],
    },

    plugins: [
      new HtmlWebpackPlugin(
        Object.assign(
          {},
          {
            title: site.title,
            inject: true,
            template: path.join(paths.source, "index.html"),
            templateParameters: site,
            favicon: path.join(paths.public, "favicon.ico"),
            chunks: ["index"],
          },
          isProd
            ? {
                minify: htmlMinify,
              }
            : {
                cache: true,
              }
        )
      ),
      new HtmlWebpackPlugin(
        Object.assign(
          {},
          {
            template: path.join(paths.source, "pages/remora.html"),
            // path: path.resolve(__dirname, paths.build, "pages"),
            filename: "pages/remora.html",
            templateParameters: site.pages.remora,
            favicon: path.join(paths.public, "favicon.ico"),
            chunks: ["remora"],
          },
          isProd
            ? {
                minify: htmlMinify,
              }
            : {
                cache: true,
              }
        )
      ),
      // new InjectImagesPlugin(),
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
            from: path.join(paths.source, "img/goofy.jpg"),
            to: path.resolve(__dirname, paths.build, "static/img/goofy.jpg"),
          },
          { from: path.join(paths.public, "robots.txt") },
          { from: path.join(paths.public, "pub.asc") },
        ],
      }),
      // new MiniCssExtractPlugin(),
      new SitemapPlugin({
        base: "https://harrybrwn.com",
        paths: sitemap,
        options: { skipgzip: false },
      }),
    ],
  };
};
