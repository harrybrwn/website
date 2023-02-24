import HTMLWebpackPlugin from "html-webpack-plugin";

type Site = {
  title: string;
  author?: string;
  domain?: string;
  description?: string;
  previewImage?: URL | string;
  github?: URL | string;
  linkedin?: URL | string;
  built: Date | string;
  og?: boolean;
  robots?: string;
  twitter?: {
    site?: string;
    creator?: string;
    card?: string;
  };
};

type HTMLMinify = {
  minifyJS: boolean;
  minifyCSS: boolean;
  collapseWhitespace: boolean;
  removeComments: boolean;
  keepClosingSlash: boolean;
  removeRedundantAttributes: boolean;
  removeStyleLinkTypeAttributes: boolean;
};

declare type HTMLOptions = {
  pageDir?: string;
  chunks?: string[];
  noChunks?: boolean;
  // filename of output destination relative to build directory
  filename?: string;
};

declare class Builder {
  paths: string;
  site: Site;
  isProd: boolean;
  htmlMinify: HTMLMinify;

  html(page: string, opts: HTMLOptions): HTMLWebpackPlugin;
  findTemplateFile(page: string, pageDir: string): string | null;
}
