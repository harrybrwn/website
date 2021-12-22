"use strict";

const common = {
  author: "Harry Brown",
  domain: "harrybrwn.com",
  description: "The home page of a humble backend software engineer.",
  previewImage: "https://harrybrwn.com/static/img/goofy.jpg",
  github: "https://github.com/harrybrwn",
  linkedin: "https://www.linkedin.com/in/harrison-brown-88823b185/",
  built: new Date(),
};

module.exports = {
  pages: {
    index: {
      title: "Harry Brown",
      ...common,
    },
    remora: {
      title: "Remora | Harry Brown",
      ...common,
    },
    blog: {
      title: "Blog | Harry Brown",
      ...common,
    },
    tanya: {
      title: "Tanya y Harry",
      ...common,
    },
  },
};
