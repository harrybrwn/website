"use strict";

const HtmlWebpackPlugin = require("html-webpack-plugin");
const path = require("path");

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

const defaultMetaTags = {
  referrer: { name: "referrer", content: "no-referrer" },
  robots: "index,archive,follow",
  googlebot: "index,archive,follow",
};

const metaTags = (site) => {
  let tags = Object.assign(
    {
      title: site.title,
      author: site.author,
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
    defaultMetaTags,
    site.subject ? { subject: site.subject } : undefined
  );

  if (site.og) {
    if (typeof site.og !== "object") {
      site.og = {};
    }
    tags = Object.assign(tags, {
      "og-url": {
        property: "og:url",
        content: site.og.url || site.og.url || `https://${site.domain}`,
      },
      "og-title": {
        property: site.og.title || site.title,
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

  if (site.twitter) {
    tags["twitter:card"] = site.twitter.card;
    tags["twitter:domain"] = site.domain;
    tags["twitter:site"] = site.twitter.site;
    tags["twitter:creator"] = site.twitter.creator || site.twitter.site;
    tags["twitter:image"] = site.twitter.image || site.previewImage;
    tags["twitter:image:src"] = site.twitter.image || site.previewImage;
    tags["twitter:description"] = site.description;
  }
  return tags;
};

const page = (paths, page, site) => {
  return {
    filename: `pages/${page}.html`,
    favicon: paths.favicon,
    template: path.join(paths.source, `pages/${page}.html`),
    templateParameters: site,
    chunks: [page.replace("-", "_")],
    meta: metaTags(site),
  };
};

module.exports = {
  metaTags,
  page,
};
