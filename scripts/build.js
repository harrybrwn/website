"use strict";

const HtmlWebpackPlugin = require("html-webpack-plugin");
const path = require("path");
const fs = require("fs");

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

// use with webpack's resolve.alias
const aliasesFromTsConfig = (tsconfig, dir) => {
  if (!tsconfig.compilerOptions) {
    return undefined;
  }
  let paths = tsconfig.compilerOptions.paths;
  if (!paths) {
    return undefined;
  }
  let result = {};
  for (let key in paths) {
    let p = path.parse(key);
    result[p.dir] = dir;
  }
  return result;
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

const metaTags = (site) => {
  let tags = Object.assign(
    {
      title: site.title,
      author: site.author,
      description: site.description,
      referrer: { name: "referrer", content: "no-referrer" },
      "og-url": { property: "og:url", content: `https://${site.domain}` },
      "og-title": { property: "og:title", content: site.title },
      "og-type": { property: "og:type", content: "website" },
      "og-description": {
        property: "og:description",
        content: site.description,
      },
      "og-image": { property: "og:image", content: site.previewImage },
      "og-locale": { property: "og:locale", content: "en_US" },
      "og-site-name": { property: "og:site_name", content: site.title },
    },
    site.subject ? { subject: site.subject } : undefined,
    site.robots ? { robots: site.robots, googlebot: site.robots } : undefined,
    site.twitter
      ? {
          "twitter:card": site.twitter.card,
          "twitter:domain": site.domain,
          "twitter:site": site.twitter.site,
          "twitter:creator": site.twitter.creator || site.twitter.site,
          "twitter:image": site.twitter.image || site.previewImage,
          "twitter:image:src": site.twitter.image || site.previewImage,
          "twitter:description": site.description,
        }
      : undefined
  );

  if (site.robots) {
    tags = Object.assign(tags, {
      robots: site.robots,
      googlebot: site.robots,
    });
  }

  if (site.og) {
    if (typeof site.og !== "object") {
      site.og = {};
    }
    tags = Object.assign(tags, {
      "og-url": {
        property: "og:url",
        content: site.og.url || `https://${site.domain}`,
      },
      "og-title": {
        property: "og:title",
        content: site.og.title || site.title,
      },
      "og-type": {
        property: "og:type",
        content: site.og.type || "website",
      },
      "og-description": {
        property: "og:description",
        content: site.og.description || site.description,
      },
      "og-image": {
        property: "og:image",
        content: site.og.image || site.previewImage,
      },
      "og-locale": {
        property: "og:locale",
        content: site.og.locale || "en_US",
      },
      "og-site-name": {
        property: "og:site_name",
        content: site.og.site_name || site.title,
      },
    });
  }
  return tags;
};

const defaultHtmlMinify = {
  minifyJS: true,
  minifyCSS: true,
  minifyURLs: true,
  collapseWhitespace: true,
  removeComments: true,
  keepClosingSlash: true,
  removeRedundantAttributes: true,
  removeStyleLinkTypeAttributes: true,
};

class Page {
  constructor(name, builder) {
    this.name = name;
    this.builder = builder;
  }
}

class Builder {
  constructor(opts) {
    this.paths = opts.paths;
    this.site = opts.site;
    this.isProd = opts.isProd;
    this.htmlMinify = opts.htmlMinify || defaultHtmlMinify;
  }

  html(page, opts) {
    if (opts === undefined) opts = {};

    if (!opts.pageDir) {
      opts.pageDir = "pages";
    }

    let chunks = [page];
    if (opts.chunks) {
      chunks = opts.chunks;
    } else if (opts.noChunks) {
      chunks = [];
    }
    // Filter out the chunk if we cant find the typescript file
    chunks = chunks.filter((val) => {
      let paths = [
        path.join("frontend", opts.pageDir, val + ".ts"),
        path.join("frontend", opts.pageDir, val, val + ".ts"),
        path.join("frontend", opts.pageDir, val, "index.ts"),
      ];
      for (let p of paths) {
        if (!fs.existsSync(p)) {
          return true;
        }
      }
      return false;
    });

    let filename;
    if (page === "index" || page === "404") {
      // Output should be in the root of the build folder
      filename = `${page}.html`;
    } else {
      filename = path.join(page, "index.html");
    }

    return new HtmlWebpackPlugin(
      Object.assign(
        {
          filename: filename,
          favicon: this.paths.favicon,
          template: this.findTemplateFile(page, opts.pageDir),
          templateParameters: this.site.pages[page],
          chunks: chunks,
          meta: metaTags(this.site.pages[page]),
        },
        this.isProd ? { minify: this.htmlMinify } : { cache: true }
      )
    );
  }

  findTemplateFile(page, pageDir) {
    let name = `${page}.html`;
    let files = [
      path.join(this.paths.source, pageDir, name),
      path.join(this.paths.source, pageDir, page, name),
      path.join(this.paths.source, pageDir, page, "index.html"),
    ];
    for (let f of files) {
      if (fs.existsSync(f)) {
        return f;
      }
    }
    return null;
  }
}

module.exports = {
  Builder,
  metaTags,
  aliasesFromTsConfig,
};
